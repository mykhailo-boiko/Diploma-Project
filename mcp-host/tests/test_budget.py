
from __future__ import annotations

import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))
from budget import SessionBudget  # noqa: E402

@pytest.mark.asyncio
async def test_increments_session_usage():
    b = SessionBudget()
    s = await b.add_usage("sess-1", "user-a", 100, session_cap=10_000)
    assert s.session_used == 100
    assert s.exceeded is False

    s2 = await b.add_usage("sess-1", "user-a", 250, session_cap=10_000)
    assert s2.session_used == 350

@pytest.mark.asyncio
async def test_marks_exceeded_when_over_cap():
    b = SessionBudget()
    s = await b.add_usage("sess-2", "user-a", 200, session_cap=150)
    assert s.exceeded is True
    assert s.reason and "session token budget" in s.reason
    assert s.suggestion and "narrow" in s.suggestion.lower()

@pytest.mark.asyncio
async def test_per_user_hour_cap():
    b = SessionBudget()
    s = await b.add_usage("sess-3", "user-b", 600, session_cap=10_000, user_hour_cap=500)
    assert s.exceeded is True
    assert "hourly" in (s.reason or "")

@pytest.mark.asyncio
async def test_negative_tokens_clamped_to_zero():
    b = SessionBudget()
    s = await b.add_usage("sess-4", "user-a", -50)
    assert s.session_used == 0

@pytest.mark.asyncio
async def test_get_status_returns_current_counters():
    b = SessionBudget()
    await b.add_usage("sess-5", "user-c", 42)
    s = await b.get_status("sess-5", "user-c")
    assert s.session_used == 42
    assert s.user_hour_used == 42

@pytest.mark.asyncio
async def test_reset_clears_session():
    b = SessionBudget()
    await b.add_usage("sess-6", "user-d", 100)
    await b.reset("sess-6")
    s = await b.get_status("sess-6")
    assert s.session_used == 0
