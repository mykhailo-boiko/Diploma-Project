
from __future__ import annotations

import time

import httpx

def test_simulator_lifecycle(gateway_url, admin_token):
    headers = {"Authorization": f"Bearer {admin_token}"}
    httpx.post(f"{gateway_url}/api/v1/simulator/stop", json={}, headers=headers, timeout=10).raise_for_status()

    start = httpx.post(
        f"{gateway_url}/api/v1/simulator/start",
        json={"scenario": "steady", "speed": 10},
        headers=headers, timeout=10,
    )
    assert start.status_code == 200
    d = start.json()["data"]
    assert d["enabled"] is True
    assert d["scenario"] == "steady"
    assert d["speed"] == 10

    status = httpx.get(f"{gateway_url}/api/v1/simulator/status", headers=headers, timeout=10).json()["data"]
    assert status["enabled"] is True

    sp = httpx.post(
        f"{gateway_url}/api/v1/simulator/speed",
        json={"speed": 5}, headers=headers, timeout=10,
    )
    assert sp.status_code == 200
    assert sp.json()["data"]["speed"] == 5

    sc = httpx.post(
        f"{gateway_url}/api/v1/simulator/scenario",
        json={"scenario": "demand_surge"}, headers=headers, timeout=10,
    )
    assert sc.status_code == 200
    assert sc.json()["data"]["scenario"] == "demand_surge"

    time.sleep(5)
    s2 = httpx.get(f"{gateway_url}/api/v1/simulator/status", headers=headers, timeout=10).json()["data"]
    c = s2["counters"]
    assert sum(c.values()) > 0, "simulator should have produced some events"

    httpx.post(f"{gateway_url}/api/v1/simulator/stop", json={}, headers=headers, timeout=10)

def test_simulator_forbidden_for_non_admin(gateway_url):
    import os
    pwd = os.getenv("OPERATOR_PASSWORD", "")
    email = os.getenv("OPERATOR_EMAIL", "")
    if not pwd or not email:
        import pytest
        pytest.skip("OPERATOR_EMAIL/PASSWORD env not provided")
    login = httpx.post(
        f"{gateway_url}/api/v1/auth/login",
        json={"email": email, "password": pwd}, timeout=10,
    )
    if login.status_code != 200:
        import pytest
        pytest.skip("operator login failed")
    tok = login.json()["data"]["access_token"]
    r = httpx.get(f"{gateway_url}/api/v1/simulator/status", headers={"Authorization": f"Bearer {tok}"}, timeout=10)
    assert r.status_code == 403, f"expected 403 for non-admin, got {r.status_code}: {r.text}"
