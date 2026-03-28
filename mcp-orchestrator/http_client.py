"""Shared HTTP client for calling Go microservices via the API gateway."""

from typing import Any

import httpx

from config import GATEWAY_URL, REQUEST_TIMEOUT

_client: httpx.AsyncClient | None = None


async def get_client() -> httpx.AsyncClient:
    """Return a shared async HTTP client (lazy-initialized)."""
    global _client
    if _client is None or _client.is_closed:
        _client = httpx.AsyncClient(
            base_url=GATEWAY_URL,
            timeout=REQUEST_TIMEOUT,
            headers={"Content-Type": "application/json"},
        )
    return _client


async def api_get(path: str, params: dict[str, Any] | None = None) -> dict[str, Any]:
    """Perform a GET request and return the JSON response."""
    client = await get_client()
    resp = await client.get(path, params=_clean_params(params))
    resp.raise_for_status()
    return resp.json()


async def api_post(path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    """Perform a POST request and return the JSON response."""
    client = await get_client()
    resp = await client.post(path, json=body)
    resp.raise_for_status()
    return resp.json()


async def api_put(path: str, body: dict[str, Any] | None = None) -> dict[str, Any]:
    """Perform a PUT request and return the JSON response."""
    client = await get_client()
    resp = await client.put(path, json=body)
    resp.raise_for_status()
    return resp.json()


async def api_delete(path: str) -> dict[str, Any]:
    """Perform a DELETE request and return the JSON response."""
    client = await get_client()
    resp = await client.delete(path)
    resp.raise_for_status()
    return resp.json()


def _clean_params(params: dict[str, Any] | None) -> dict[str, str] | None:
    """Remove None values from query parameters and convert to strings."""
    if params is None:
        return None
    return {k: str(v) for k, v in params.items() if v is not None}
