
from __future__ import annotations

import asyncio
import logging
from typing import Any

import httpx

from config import (
    GATEWAY_URL,
    REQUEST_TIMEOUT,
    SERVICE_USER_EMAIL,
    SERVICE_USER_PASSWORD,
)

logger = logging.getLogger(__name__)

_client: httpx.AsyncClient | None = None
_token: str | None = None
_trace_id: str | None = None
_token_lock = asyncio.Lock()

async def get_client() -> httpx.AsyncClient:
    global _client
    if _client is None or _client.is_closed:
        _client = httpx.AsyncClient(
            base_url=GATEWAY_URL,
            timeout=REQUEST_TIMEOUT,
            headers={"Content-Type": "application/json"},
        )
    return _client

def set_trace_id(trace_id: str | None) -> None:

    global _trace_id
    _trace_id = trace_id

def get_trace_id() -> str | None:
    return _trace_id

async def _login() -> str | None:
    if not SERVICE_USER_EMAIL or not SERVICE_USER_PASSWORD:
        return None
    client = await get_client()
    resp = await client.post(
        "/api/v1/auth/login",
        json={"email": SERVICE_USER_EMAIL, "password": SERVICE_USER_PASSWORD},
    )
    if resp.status_code != 200:
        return None
    data = resp.json()
    return (data.get("data") or {}).get("access_token")

async def _ensure_token(force: bool = False) -> str | None:
    global _token
    async with _token_lock:
        if _token is None or force:
            _token = await _login()
        return _token

async def _auth_headers() -> dict[str, str]:
    token = await _ensure_token()
    headers: dict[str, str] = {}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    if _trace_id:
        headers["X-Trace-ID"] = _trace_id
    return headers

async def _request(method: str, path: str, **kwargs: Any) -> httpx.Response:
    client = await get_client()
    headers = await _auth_headers()
    resp = await client.request(method, path, headers=headers, **kwargs)
    if resp.status_code == 401 and SERVICE_USER_EMAIL and SERVICE_USER_PASSWORD:
        token = await _ensure_token(force=True)
        if token:
            retry_headers = {"Authorization": f"Bearer {token}"}
            if _trace_id:
                retry_headers["X-Trace-ID"] = _trace_id
            resp = await client.request(method, path, headers=retry_headers, **kwargs)
    return resp

_STATUS_HINTS: dict[int, dict[str, str]] = {
    400: {
        "code": "validation_error",
        "suggestion": "Check the parameter types and required fields against the tool docstring and try again with corrected arguments.",
    },
    401: {
        "code": "auth_required",
        "suggestion": "The session token may have expired. This is an internal authentication issue; report to the user rather than retrying.",
    },
    403: {
        "code": "forbidden",
        "suggestion": "The current user role lacks permission for this operation. Inform the user that admin (or higher-privileged) access is required.",
    },
    404: {
        "code": "not_found",
        "suggestion": "Verify the entity ID exists by calling the corresponding *_list or *_get tool first. The ID may be incorrect, deleted, or belong to a different scope.",
    },
    409: {
        "code": "conflict",
        "suggestion": "The operation conflicts with the current state (e.g. invalid status transition). Fetch the current state and adjust the request.",
    },
    422: {
        "code": "unprocessable_entity",
        "suggestion": "The request shape is correct but values are semantically invalid. Re-read the error details and fix the offending fields.",
    },
    429: {
        "code": "rate_limit_exceeded",
        "suggestion": "Slow down. Wait before retrying or batch fewer items per request.",
    },
    500: {
        "code": "internal_error",
        "suggestion": "Server-side error — likely not caused by tool arguments. Do NOT retry blindly; report the error to the user.",
    },
    502: {
        "code": "bad_gateway",
        "suggestion": "Upstream service unreachable. Do not retry immediately; report to the user.",
    },
    503: {
        "code": "service_unavailable",
        "suggestion": "Backend service is starting up or under maintenance. Do not retry within this turn.",
    },
    504: {
        "code": "timeout",
        "suggestion": "Backend timed out. Reduce the result set (lower limit / narrower filter) and retry once.",
    },
}

def _wrap_error(method: str, path: str, resp: httpx.Response) -> dict[str, Any]:
    status = resp.status_code
    hint = _STATUS_HINTS.get(status, {
        "code": f"http_{status}",
        "suggestion": "Unexpected HTTP status. Report the error to the user with the message field.",
    })
    err: dict[str, Any] = {
        "code": hint["code"],
        "message": f"{method} {path} failed with HTTP {status}",
        "http_status": status,
        "suggestion": hint["suggestion"],
    }
    try:
        body = resp.json()
        if isinstance(body, dict):
            backend = body.get("error")
            if isinstance(backend, dict):
                if backend.get("code"):
                    err["code"] = backend["code"]
                if backend.get("message"):
                    err["message"] = backend["message"]
                data = backend.get("data") or {}
                for k in ("field", "expected", "received", "suggestion", "examples", "doc_ref"):
                    if k in data and data[k] is not None:
                        err[k] = data[k]
    except Exception:
        if resp.text:
            err["received_body"] = resp.text[:500]
    logger.warning("HTTP %s %s -> %s (%s)", method, path, status, err.get("code"))
    return {"error": err}

async def _run(method: str, path: str, **kwargs: Any) -> dict[str, Any]:
    try:
        resp = await _request(method, path, **kwargs)
    except httpx.TimeoutException as exc:
        return {"error": {
            "code": "request_timeout",
            "message": f"{method} {path} timed out after {REQUEST_TIMEOUT}s",
            "suggestion": "The backend is slow. Narrow the filter or lower the page limit and retry once.",
            "received": str(exc),
        }}
    except httpx.HTTPError as exc:
        return {"error": {
            "code": "transport_error",
            "message": f"{method} {path}: {exc.__class__.__name__}",
            "suggestion": "Network/transport failure between MCP and the API gateway. Do not retry blindly.",
            "received": str(exc),
        }}
    if resp.status_code >= 400:
        return _wrap_error(method, path, resp)
    if not resp.content:
        return {}
    try:
        return resp.json()
    except Exception as exc:
        return {"error": {
            "code": "invalid_json_response",
            "message": f"{method} {path}: response was not valid JSON",
            "suggestion": "This indicates a backend bug; report to the user with the message field.",
            "received": str(exc),
        }}

async def api_get(path: str, params: dict[str, Any] | None = None) -> dict[str, Any]:
    return await _run("GET", path, params=_clean_params(params))

async def api_get_all(
    path: str,
    params: dict[str, Any] | None = None,
    page_size: int = 1000,
    hard_cap: int = 5000,
) -> dict[str, Any]:

    base = dict(params or {})
    base["limit"] = page_size
    aggregated: list[Any] = []
    meta: dict[str, Any] = {}
    offset = 0
    for _ in range(50):
        page_params = {**base, "offset": offset}
        body = await _run("GET", path, params=_clean_params(page_params))
        if isinstance(body, dict) and "error" in body and "data" not in body:
            return body
        data = body.get("data") if isinstance(body, dict) else None
        if not isinstance(data, list):
            return body
        aggregated.extend(data)
        meta = body.get("meta") or meta
        if len(data) < page_size or len(aggregated) >= hard_cap:
            break
        offset += page_size
    return {
        "data": aggregated[:hard_cap],
        "meta": {**meta, "total_returned": len(aggregated)},
    }

async def api_post(path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    return await _run("POST", path, json=_clean_body(body))

async def api_put(path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    return await _run("PUT", path, json=_clean_body(body))

async def api_patch(path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    return await _run("PATCH", path, json=_clean_body(body))

def _clean_body(body: dict[str, Any] | None) -> dict[str, Any] | None:

    if body is None:
        return None
    from datetime import date, datetime

    def cast(v: Any) -> Any:
        if isinstance(v, datetime):
            return v.isoformat().replace("+00:00", "Z")
        if isinstance(v, date):
            return v.isoformat()
        if isinstance(v, dict):
            return {k: cast(x) for k, x in v.items()}
        if isinstance(v, list):
            return [cast(x) for x in v]
        return v

    return cast(body)

async def api_delete(path: str) -> dict[str, Any]:
    return await _run("DELETE", path)

def _clean_params(params: dict[str, Any] | None) -> dict[str, str] | None:
    if params is None:
        return None
    from datetime import date, datetime

    out: dict[str, str] = {}
    for k, v in params.items():
        if v is None:
            continue
        if isinstance(v, datetime):
            out[k] = v.isoformat().replace("+00:00", "Z")
        elif isinstance(v, date):
            out[k] = v.isoformat()
        elif isinstance(v, bool):
            out[k] = "true" if v else "false"
        else:
            out[k] = str(v)
    return out
