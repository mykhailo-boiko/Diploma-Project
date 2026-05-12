
from __future__ import annotations

import json
import threading
import time

import httpx

def test_sse_receives_events_when_simulator_runs(gateway_url, admin_token):
    events: list[dict] = []

    def consume() -> None:
        with httpx.stream(
            "GET",
            f"{gateway_url}/api/v1/events/stream",
            headers={"Authorization": f"Bearer {admin_token}", "Accept": "text/event-stream"},
            timeout=httpx.Timeout(60.0, read=60.0),
        ) as r:
            data_buf = ""
            for line in r.iter_lines():
                if line is None:
                    continue
                if isinstance(line, bytes):
                    line = line.decode("utf-8", "ignore")
                if line.startswith("data:"):
                    data_buf = line[5:].strip()
                elif line == "" and data_buf:
                    try:
                        events.append(json.loads(data_buf))
                    except Exception:
                        pass
                    data_buf = ""
                if len(events) >= 5:
                    return

    t = threading.Thread(target=consume, daemon=True)
    t.start()

    headers = {"Authorization": f"Bearer {admin_token}"}
    httpx.post(
        f"{gateway_url}/api/v1/simulator/start",
        json={"scenario": "steady", "speed": 25},
        headers=headers, timeout=10.0,
    )

    try:
        t.join(timeout=35.0)
    finally:
        httpx.post(f"{gateway_url}/api/v1/simulator/stop", json={}, headers=headers, timeout=10.0)

    assert len(events) >= 2, f"expected at least 2 SSE events, got {len(events)}: {events!r}"
    types = {e.get("type") for e in events}
    assert types - {"connection.established"}, "must see business events, not only connection.established"
