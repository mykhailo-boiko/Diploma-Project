
from __future__ import annotations

import asyncio
import json
import os
import time
import uuid

import pytest

import httpx
import websockets

@pytest.mark.skipif(os.getenv("E2E_LLM") != "1", reason="E2E_LLM=1 required to spend Gemini quota")
def test_chat_simple_query(gateway_url, admin_token):
    mcp_ws_base = os.getenv("MCP_WS_URL", "ws://localhost:8090/ws/chat")
    url = f"{mcp_ws_base}?token={admin_token}"

    async def run():
        results: list[dict] = []
        async with websockets.connect(url) as ws:
            await ws.send(json.dumps({"message": "How many orders are in the system?",
                                       "trace_id": f"e2e-{uuid.uuid4()}"}))
            try:
                while True:
                    raw = await asyncio.wait_for(ws.recv(), timeout=90.0)
                    msg = json.loads(raw)
                    results.append(msg)
                    if msg.get("type") == "message":
                        return results
            except asyncio.TimeoutError:
                return results

    out = asyncio.run(run())
    types = [m.get("type") for m in out]
    assert "tool_start" in types or "tool_result" in types, f"expected at least one tool call: {types}"
    assert "message" in types, f"expected final message: {types}"
