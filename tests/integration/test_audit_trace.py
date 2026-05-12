
from __future__ import annotations

import time

def test_audit_trace_by_entity(api, pick_first_product):
    o = api.post("/api/v1/orders", json={
        "customer_name": "Trace Tester",
        "items": [{"product_id": pick_first_product["id"], "name": pick_first_product["name"],
                   "quantity": 1, "unit_price": pick_first_product["unit_price"]}],
    }).json()["data"]
    order_id = o["id"]

    api.put(f"/api/v1/orders/{order_id}/status", json={"status": "confirmed"})
    api.put(f"/api/v1/orders/{order_id}/status", json={"status": "processing"})

    time.sleep(2)
    trace = api.get("/api/v1/analytics/trace/by-entity", params={"entity_id": order_id})
    assert trace.status_code == 200, trace.text
    body = trace.json()["data"]
    assert body["entity_id"] == order_id
    assert body["total"] >= 1
    events = body["events"]
    actions = [e["action"] for e in events]
    assert any("orders.create" in a for a in actions), f"expected orders.create in {actions}"

def test_audit_log_query_by_action(api):
    resp = api.get("/api/v1/analytics/audit-log", params={"action": "orders.create", "limit": "5"})
    assert resp.status_code == 200
    rows = resp.json()["data"]
    assert isinstance(rows, list)
    for r in rows:
        assert r["action"] == "orders.create"

def test_audit_trace_missing_entity_id_returns_validation_error(api):
    r = api.get("/api/v1/analytics/trace/by-entity")
    assert r.status_code == 400
    err = r.json()["error"]
    assert "field" in (err.get("data") or {}) or err["code"] in {"missing_field", "validation_error"}
