
from __future__ import annotations

import time

def test_order_lifecycle_pending_to_delivered(api, pick_first_product):
    p = pick_first_product
    body = {
        "customer_name": "Integration Tester",
        "items": [{
            "product_id": p["id"],
            "name": p["name"],
            "quantity": 2,
            "unit_price": p["unit_price"],
        }],
    }
    create = api.post("/api/v1/orders", json=body)
    assert create.status_code == 201, create.text
    order = create.json()["data"]
    order_id = order["id"]
    assert order["status"] == "pending"
    assert order["total_amount"] >= 2 * p["unit_price"]

    for next_status in ("confirmed", "processing", "shipped", "delivered"):
        resp = api.put(f"/api/v1/orders/{order_id}/status", json={"status": next_status})
        assert resp.status_code == 200, f"transition→{next_status} failed: {resp.text}"
        body = resp.json()["data"]
        assert body["status"] == next_status

    final = api.get(f"/api/v1/orders/{order_id}").json()["data"]
    assert final["status"] == "delivered"

    time.sleep(1.5)
    audit = api.get("/api/v1/analytics/trace/by-entity", params={"entity_id": order_id})
    assert audit.status_code == 200
    body = audit.json()["data"]
    actions = {ev["action"] for ev in body["events"]}
    assert "orders.create" in actions
    assert any(a.startswith("orders.update_status") or "status" in a for a in actions)

def test_order_invalid_transition_returns_structured_error(api, pick_first_product):
    p = pick_first_product
    create = api.post("/api/v1/orders", json={
        "customer_name": "Tester Invalid",
        "items": [{"product_id": p["id"], "name": p["name"], "quantity": 1, "unit_price": p["unit_price"]}],
    })
    order_id = create.json()["data"]["id"]

    bad = api.put(f"/api/v1/orders/{order_id}/status", json={"status": "delivered"})
    assert bad.status_code in (400, 409), bad.text
    err = bad.json().get("error", {})
    assert err.get("code") in {"invalid_transition", "invalid_status_transition"}, err
    if "data" in err and err["data"]:
        d = err["data"]
        assert "suggestion" in d, "structured error must include suggestion for LLM self-correction"

def test_order_cancel_with_reason(api, pick_first_product):
    p = pick_first_product
    create = api.post("/api/v1/orders", json={
        "customer_name": "Cancel Tester",
        "items": [{"product_id": p["id"], "name": p["name"], "quantity": 1, "unit_price": p["unit_price"]}],
    })
    order_id = create.json()["data"]["id"]
    resp = api.post(f"/api/v1/orders/{order_id}/cancel", json={"reason": "integration test"})
    assert resp.status_code == 200, resp.text
    assert resp.json()["data"]["status"] == "cancelled"
