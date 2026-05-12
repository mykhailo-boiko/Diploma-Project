
from __future__ import annotations

def test_invalid_date_format_returns_structured_error(api):
    r = api.get("/api/v1/analytics/sales", params={"date_from": "2026/05/01", "date_to": "2026-05-15"})
    assert r.status_code in (400, 422), r.text

def test_missing_required_field_returns_structured_error(api):
    r = api.post("/api/v1/orders", json={"customer_name": ""})
    assert r.status_code == 400, r.text
    err = r.json()["error"]
    assert err["code"] in {"missing_field", "validation_error"}, err
    data = err.get("data", {}) or {}
    assert data.get("suggestion"), "validation error must include suggestion"
    if "field" in data:
        assert data["field"] in {"customer_name", "items"}

def test_invalid_stock_adjustment_type(api, pick_first_product, pick_first_warehouse):
    r = api.post("/api/v1/stock/adjust", json={
        "product_id": pick_first_product["id"],
        "warehouse_id": pick_first_warehouse["id"],
        "quantity": 1,
        "type": "bogus_type",
        "reference": "test",
    })
    assert r.status_code == 400
    err = r.json()["error"]
    assert err["code"] in {"invalid_field", "validation_error"}, err
    data = err.get("data", {}) or {}
    if "examples" in data:
        assert "inbound" in data["examples"]

def test_negative_stock_quantity_rejected(api, pick_first_product, pick_first_warehouse):
    r = api.post("/api/v1/stock/adjust", json={
        "product_id": pick_first_product["id"],
        "warehouse_id": pick_first_warehouse["id"],
        "quantity": -5,
        "type": "outbound",
        "reference": "test",
    })
    assert r.status_code == 400
    err = r.json()["error"]
    data = err.get("data", {}) or {}
    if data:
        assert "suggestion" in data
