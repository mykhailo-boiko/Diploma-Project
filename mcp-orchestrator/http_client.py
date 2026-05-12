"""Shared HTTP client for calling Go microservices via the API gateway.

Authenticates as a service user on first use and re-authenticates if a 401
response is received. This makes MCP tool calls work transparently as the
configured service identity (typically admin)."""

from __future__ import annotations

import asyncio
from typing import Any

import httpx

from config import (
    GATEWAY_URL,
    REQUEST_TIMEOUT,
    SERVICE_USER_EMAIL,
    SERVICE_USER_PASSWORD,
)

_client: httpx.AsyncClient | None = None
_token: str | None = None
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
    return {"Authorization": f"Bearer {token}"} if token else {}


async def _request(method: str, path: str, **kwargs: Any) -> httpx.Response:
    client = await get_client()
    headers = await _auth_headers()
    resp = await client.request(method, path, headers=headers, **kwargs)
    if resp.status_code == 401 and SERVICE_USER_EMAIL and SERVICE_USER_PASSWORD:
        token = await _ensure_token(force=True)
        if token:
            resp = await client.request(
                method, path,
                headers={"Authorization": f"Bearer {token}"},
                **kwargs,
            )
    return resp


async def api_get(path: str, params: dict[str, Any] | None = None) -> dict[str, Any]:
    resp = await _request("GET", path, params=_clean_params(params))
    resp.raise_for_status()
    return resp.json()


async def api_get_all(
    path: str,
    params: dict[str, Any] | None = None,
    page_size: int = 1000,
    hard_cap: int = 5000,
) -> dict[str, Any]:
    """Fetch all pages from a paginated list endpoint and return one combined response.

    Iterates with limit=page_size + offset until either:
      - the API returns fewer items than page_size (last page),
      - cumulative items reach hard_cap, or
      - 50 pages have been requested (safety).
    The response shape mirrors a single api_get response: {"data": [...], "meta": {...}}.
    """
    base = dict(params or {})
    base["limit"] = page_size
    aggregated: list[Any] = []
    meta: dict[str, Any] = {}
    offset = 0
    for _ in range(50):
        page_params = {**base, "offset": offset}
        resp = await _request("GET", path, params=_clean_params(page_params))
        resp.raise_for_status()
        body = resp.json() if resp.content else {}
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
    resp = await _request("POST", path, json=body)
    resp.raise_for_status()
    return resp.json()


async def api_put(path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    resp = await _request("PUT", path, json=body)
    resp.raise_for_status()
    return resp.json()


async def api_patch(path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    resp = await _request("PATCH", path, json=body)
    resp.raise_for_status()
    return resp.json()


async def api_delete(path: str) -> dict[str, Any]:
    resp = await _request("DELETE", path)
    resp.raise_for_status()
    return resp.json()


def _clean_params(params: dict[str, Any] | None) -> dict[str, str] | None:
    if params is None:
        return None
    return {k: str(v) for k, v in params.items() if v is not None}
