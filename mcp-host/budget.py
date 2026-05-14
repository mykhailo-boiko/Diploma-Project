
from __future__ import annotations

import logging
import os
from dataclasses import dataclass
from typing import Any

logger = logging.getLogger(__name__)

DEFAULT_SESSION_CAP = int(os.getenv("MAX_TOKENS_PER_SESSION", "200000"))
DEFAULT_USER_HOUR_CAP = int(os.getenv("MAX_TOKENS_PER_USER_PER_HOUR", "1000000"))
DEFAULT_SESSION_TTL = int(os.getenv("SESSION_TTL", "1800"))
HOUR_WINDOW_SEC = 3600

@dataclass
class BudgetStatus:
    session_used: int
    session_cap: int
    user_hour_used: int
    user_hour_cap: int
    exceeded: bool
    reason: str | None
    suggestion: str | None

class SessionBudget:
    def __init__(self, redis: Any | None = None) -> None:
        self._redis = redis
        self._fallback: dict[str, int] = {}

    def attach(self, redis: Any) -> None:
        self._redis = redis

    @staticmethod
    def _session_key(session_id: str) -> str:
        return f"budget:session:{session_id}"

    @staticmethod
    def _user_hour_key(user_id: str) -> str:
        return f"budget:user:{user_id}:hour"

    async def add_usage(
        self,
        session_id: str,
        user_id: str | None,
        tokens: int,
        *,
        session_cap: int = DEFAULT_SESSION_CAP,
        user_hour_cap: int = DEFAULT_USER_HOUR_CAP,
    ) -> BudgetStatus:
        if tokens < 0:
            tokens = 0

        session_used = await self._incr(self._session_key(session_id), tokens, DEFAULT_SESSION_TTL)
        user_hour_used = 0
        if user_id:
            user_hour_used = await self._incr(self._user_hour_key(user_id), tokens, HOUR_WINDOW_SEC)

        exceeded = False
        reason: str | None = None
        suggestion: str | None = None
        if session_used >= session_cap:
            exceeded = True
            reason = (
                f"session token budget exhausted: {session_used} >= cap {session_cap}. "
                "Further LLM/tool calls would amplify cost without bounded progress."
            )
            suggestion = (
                "Tell the user that the request is too large to complete in one chat. "
                "Ask the user to send /clear to reset the token budget for this session, "
                "then narrow the question (smaller date window, fewer entities, more specific filter)."
            )
        elif user_id and user_hour_used >= user_hour_cap:
            exceeded = True
            reason = (
                f"per-user hourly token budget exhausted: {user_hour_used} >= cap {user_hour_cap} "
                "in the last hour."
            )
            suggestion = (
                "Tell the user they have exceeded the hourly quota for AI usage. "
                "The quota resets one hour after the first call in the current window."
            )

        return BudgetStatus(
            session_used=session_used,
            session_cap=session_cap,
            user_hour_used=user_hour_used,
            user_hour_cap=user_hour_cap,
            exceeded=exceeded,
            reason=reason,
            suggestion=suggestion,
        )

    async def get_status(
        self,
        session_id: str,
        user_id: str | None = None,
        *,
        session_cap: int = DEFAULT_SESSION_CAP,
        user_hour_cap: int = DEFAULT_USER_HOUR_CAP,
    ) -> BudgetStatus:
        session_used = await self._get(self._session_key(session_id))
        user_hour_used = await self._get(self._user_hour_key(user_id)) if user_id else 0
        return BudgetStatus(
            session_used=session_used,
            session_cap=session_cap,
            user_hour_used=user_hour_used,
            user_hour_cap=user_hour_cap,
            exceeded=session_used >= session_cap or (user_id is not None and user_hour_used >= user_hour_cap),
            reason=None,
            suggestion=None,
        )

    async def reset(self, session_id: str) -> None:
        if self._redis is not None:
            try:
                await self._redis.delete(self._session_key(session_id))
            except Exception as exc:
                logger.debug("budget reset failed: %s", exc)
        self._fallback.pop(self._session_key(session_id), None)

    async def _incr(self, key: str, delta: int, ttl: int) -> int:
        if self._redis is None:
            current = self._fallback.get(key, 0) + delta
            self._fallback[key] = current
            return current
        try:
            new_val = await self._redis.incrby(key, delta)
            await self._redis.expire(key, ttl, nx=True)
            return int(new_val)
        except Exception as exc:
            logger.warning("budget incr failed for %s: %s", key, exc)
            current = self._fallback.get(key, 0) + delta
            self._fallback[key] = current
            return current

    async def _get(self, key: str) -> int:
        if self._redis is None:
            return self._fallback.get(key, 0)
        try:
            val = await self._redis.get(key)
            return int(val) if val else 0
        except Exception:
            return self._fallback.get(key, 0)
