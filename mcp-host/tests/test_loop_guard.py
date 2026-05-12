
from __future__ import annotations

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from loop_guard import LoopGuard, THRESHOLD  # noqa: E402

@pytest.mark.asyncio
async def test_no_loop_for_unique_calls():
    g = LoopGuard()
    r = await g.check("s1", "orders_list", {"limit": 100})
    assert r.loop is False

    r = await g.check("s1", "orders_get", {"order_id": "abc"})
    assert r.loop is False

@pytest.mark.asyncio
async def test_detects_loop_when_threshold_reached():
    g = LoopGuard()
    sig_args = {"order_id": "same-id"}
    for _ in range(THRESHOLD - 1):
        r = await g.check("s2", "orders_get", sig_args)
        assert r.loop is False, "should not flag before threshold"
    r = await g.check("s2", "orders_get", sig_args)
    assert r.loop is True
    assert r.occurrences >= THRESHOLD
    assert r.message and "loop detected" in r.message
    assert r.suggestion and "different" in r.suggestion.lower()

@pytest.mark.asyncio
async def test_args_hash_distinguishes_different_calls():
    g = LoopGuard()
    for i in range(5):
        r = await g.check("s3", "orders_list", {"limit": i})
        assert r.loop is False, f"different args should not loop ({i})"

@pytest.mark.asyncio
async def test_reset_clears_history():
    g = LoopGuard()
    for _ in range(THRESHOLD):
        await g.check("s4", "tool_x", {})
    await g.reset("s4")
    r = await g.check("s4", "tool_x", {})
    assert r.loop is False

@pytest.mark.asyncio
async def test_session_isolation():
    g = LoopGuard()
    for _ in range(THRESHOLD):
        await g.check("session-A", "tool_y", {})
    r = await g.check("session-B", "tool_y", {})
    assert r.loop is False, "loop in one session should not affect another"
