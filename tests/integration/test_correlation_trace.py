
from __future__ import annotations

import time
import uuid

def test_trace_id_propagates_to_audit_log(api, pick_first_product):
    trace_id = f"itest-{uuid.uuid4()}"
    p = pick_first_product
    resp = api._client.post(
        "/api/v1/orders",
        json={
            "customer_name": "Trace Propagation",
            "items": [{"product_id": p["id"], "name": p["name"], "quantity": 1, "unit_price": p["unit_price"]}],
        },
        headers={"X-Trace-ID": trace_id},
    )
    assert resp.status_code == 201, resp.text
    order_id = resp.json()["data"]["id"]

    time.sleep(2)
    trace = api.get("/api/v1/analytics/trace/by-entity", params={"entity_id": order_id}).json()["data"]
    trace_ids = trace.get("trace_ids") or []
    if not trace_ids:
        for ev in trace["events"]:
            if ev.get("trace_id"):
                trace_ids.append(ev["trace_id"])
    assert trace_id in trace_ids, f"expected {trace_id} in audit trace_ids: {trace_ids}"
