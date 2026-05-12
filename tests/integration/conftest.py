
from __future__ import annotations

import os
import time
from typing import Any

import httpx
import pytest

GATEWAY_URL = os.getenv("GATEWAY_URL", "http://localhost:8080")
MCP_WS_URL = os.getenv("MCP_WS_URL", "ws://localhost:8090/ws/chat")
ADMIN_EMAIL = os.getenv("ADMIN_EMAIL", "admin@chainorchestra.local")
ADMIN_PASSWORD = os.getenv("ADMIN_PASSWORD", "")

def _login(email: str, password: str) -> str:
    if not password:
        pytest.skip("ADMIN_PASSWORD env not set")
    resp = httpx.post(
        f"{GATEWAY_URL}/api/v1/auth/login",
        json={"email": email, "password": password},
        timeout=15.0,
    )
    resp.raise_for_status()
    return resp.json()["data"]["access_token"]

@pytest.fixture(scope="session")
def admin_token() -> str:
    return _login(ADMIN_EMAIL, ADMIN_PASSWORD)

@pytest.fixture(scope="session")
def gateway_url() -> str:
    return GATEWAY_URL

@pytest.fixture(scope="session")
def mcp_ws_url() -> str:
    return MCP_WS_URL

class APIClient:
    def __init__(self, base_url: str, token: str) -> None:
        self.base_url = base_url
        self._token = token
        self._client = httpx.Client(
            base_url=base_url,
            timeout=30.0,
            headers={
                "Authorization": f"Bearer {token}",
                "Content-Type": "application/json",
            },
        )

    def get(self, path: str, **kwargs: Any) -> httpx.Response:
        return self._client.get(path, **kwargs)

    def post(self, path: str, json: dict | None = None, **kwargs: Any) -> httpx.Response:
        return self._client.post(path, json=json, **kwargs)

    def put(self, path: str, json: dict | None = None, **kwargs: Any) -> httpx.Response:
        return self._client.put(path, json=json, **kwargs)

    def patch(self, path: str, json: dict | None = None, **kwargs: Any) -> httpx.Response:
        return self._client.patch(path, json=json, **kwargs)

    def delete(self, path: str, **kwargs: Any) -> httpx.Response:
        return self._client.delete(path, **kwargs)

    def close(self) -> None:
        self._client.close()

@pytest.fixture()
def api(gateway_url: str, admin_token: str):
    client = APIClient(gateway_url, admin_token)
    try:
        yield client
    finally:
        client.close()

@pytest.fixture()
def pick_first_active_carrier(api: APIClient) -> dict:
    resp = api.get("/api/v1/carriers", params={"is_active": "true", "limit": "1"})
    resp.raise_for_status()
    items = resp.json()["data"]
    if not items:
        pytest.skip("no active carrier available")
    return items[0]

@pytest.fixture()
def pick_first_warehouse(api: APIClient) -> dict:
    resp = api.get("/api/v1/warehouses", params={"limit": "1"})
    resp.raise_for_status()
    items = resp.json()["data"]
    if not items:
        pytest.skip("no warehouse available")
    return items[0]

@pytest.fixture()
def pick_first_product(api: APIClient) -> dict:
    resp = api.get("/api/v1/products", params={"limit": "1"})
    resp.raise_for_status()
    items = resp.json()["data"]
    if not items:
        pytest.skip("no product available")
    return items[0]

def wait_for_condition(predicate, *, timeout: float = 30.0, interval: float = 1.0, desc: str = "condition") -> bool:
    start = time.time()
    while time.time() - start < timeout:
        try:
            if predicate():
                return True
        except Exception:
            pass
        time.sleep(interval)
    return False

pytest_plugins: list[str] = []
