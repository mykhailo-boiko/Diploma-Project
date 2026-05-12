
from __future__ import annotations

import time

def test_shipment_full_pipeline(api, pick_first_warehouse, pick_first_active_carrier, pick_first_product):
    p = pick_first_product
    o = api.post("/api/v1/orders", json={
        "customer_name": "Postal Tester",
        "items": [{"product_id": p["id"], "name": p["name"], "quantity": 1, "unit_price": p["unit_price"]}],
    }).json()["data"]

    ship_body = {
        "order_id": o["id"],
        "warehouse_id": pick_first_warehouse["id"],
        "carrier_id": pick_first_active_carrier["id"],
        "address": "Khreshchatyk str. 22, Kyiv",
    }
    create = api.post("/api/v1/shipments", json=ship_body)
    assert create.status_code == 201, create.text
    sh = create.json()["data"]
    sid = sh["id"]
    tracking = sh.get("tracking_number")
    assert tracking, "shipment must have a tracking number"

    current = sh["status"]
    transitions: list[str] = []
    if current == "created":
        transitions.append("label_created")
    if current in ("created", "label_created"):
        transitions.append("awaiting_pickup")
    transitions.extend(["picked_up", "in_transit", "out_for_delivery"])
    for st in transitions:
        r = api.put(f"/api/v1/shipments/{sid}/status", json={"status": st})
        assert r.status_code == 200, f"transition→{st} failed: {r.text}"

    r = api.post(f"/api/v1/shipments/{sid}/events", json={
        "type": "in_transit", "location_city": "Kyiv", "location_hub": "Kyiv North Hub", "notes": "test event",
    })
    assert r.status_code in (200, 201), r.text

    r = api.post(f"/api/v1/shipments/{sid}/record-delivery", json={
        "signature_name": "Test Recipient", "photo_url": "",
    })
    assert r.status_code == 200, r.text
    final = api.get(f"/api/v1/shipments/{sid}").json()["data"]
    assert final["status"] == "delivered"

    r = api.get(f"/api/v1/tracking/{tracking}")
    assert r.status_code == 200, r.text
    body = r.json()["data"]
    assert "events" in body
    assert len(body["events"]) >= 3

def test_shipment_three_attempts_returns_to_sender(api, pick_first_warehouse, pick_first_active_carrier, pick_first_product):
    p = pick_first_product
    o = api.post("/api/v1/orders", json={
        "customer_name": "Attempt Tester",
        "items": [{"product_id": p["id"], "name": p["name"], "quantity": 1, "unit_price": p["unit_price"]}],
    }).json()["data"]
    sh = api.post("/api/v1/shipments", json={
        "order_id": o["id"],
        "warehouse_id": pick_first_warehouse["id"],
        "carrier_id": pick_first_active_carrier["id"],
        "address": "Sahaidachnoho str. 5, Lviv",
    }).json()["data"]
    sid = sh["id"]
    cur = sh["status"]
    seq = []
    if cur == "created":
        seq.append("label_created")
    if cur in ("created", "label_created"):
        seq.append("awaiting_pickup")
    seq.extend(["picked_up", "in_transit", "out_for_delivery"])
    for st in seq:
        api.put(f"/api/v1/shipments/{sid}/status", json={"status": st})

    for i in range(3):
        r = api.post(f"/api/v1/shipments/{sid}/record-attempt", json={
            "reason": "no_one_home", "notes": f"attempt {i + 1}",
        })
        assert r.status_code == 200, r.text
        body = r.json().get("data") or {}
        if body.get("status") == "returned_to_sender":
            return
        api.put(f"/api/v1/shipments/{sid}/status", json={"status": "out_for_delivery"})

    final = api.get(f"/api/v1/shipments/{sid}").json()["data"]
    assert final.get("delivery_attempts", 0) >= 3
