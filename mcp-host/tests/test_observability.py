
from __future__ import annotations

import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from observability import extract_entity_ids, logfire_span, setup_logfire  # noqa: E402

def test_extract_entity_ids_collects_known_keys():
    args = {
        "order_id": "abc",
        "shipment_id": "def",
        "limit": 100,
        "customer_name": "Foo Bar",
    }
    out = extract_entity_ids(args)
    assert out == {"order_id": "abc", "shipment_id": "def", "customer_name": "Foo Bar"}

def test_extract_entity_ids_handles_empty_or_invalid():
    assert extract_entity_ids({}) == {}
    assert extract_entity_ids(None) == {}
    assert extract_entity_ids({"limit": 10}) == {}

def test_logfire_span_is_noop_without_token(monkeypatch):
    monkeypatch.delenv("LOGFIRE_TOKEN", raising=False)
    setup_logfire("test-svc")
    with logfire_span("test.span", k=1) as span:
        span.set_attribute("foo", "bar")
        span.set_attributes(x=1, y=2)
