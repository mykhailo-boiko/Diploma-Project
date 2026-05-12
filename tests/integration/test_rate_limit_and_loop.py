
from __future__ import annotations

import httpx

def test_concurrent_requests_dont_break_gateway(api):
    statuses: list[int] = []
    for _ in range(50):
        r = api.get("/api/v1/orders/stats")
        statuses.append(r.status_code)
    assert all(s == 200 for s in statuses[:5]), f"first batch failures: {statuses[:5]}"
    assert statuses.count(200) >= 40, f"too many failures: {statuses}"

def test_budget_endpoint_returns_status(gateway_url, admin_token):
    import os
    mcp_host_url = os.getenv("MCP_HOST_URL", "http://localhost:8090")
    r = httpx.get(f"{mcp_host_url}/api/v1/mcp/budget/test-session", timeout=10)
    assert r.status_code == 200
    body = r.json()
    assert "session_used" in body
    assert "session_cap" in body
    assert body["session_cap"] > 0
