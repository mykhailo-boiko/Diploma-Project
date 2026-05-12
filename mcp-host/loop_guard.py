
from __future__ import annotations

import hashlib
import json
import logging
import time
from dataclasses import dataclass
from typing import Any

logger = logging.getLogger(__name__)

WINDOW = 8
THRESHOLD = 3
KEY_TTL = 600

@dataclass
class LoopCheck:
    loop: bool
    occurrences: int
    suggestion: str | None
    message: str | None

def _args_hash(arguments: dict[str, Any]) -> str:
    try:
        canon = json.dumps(arguments or {}, sort_keys=True, default=str)
    except Exception:
        canon = repr(arguments)
    return hashlib.sha256(canon.encode()).hexdigest()[:16]

class LoopGuard:
    def __init__(self, redis: Any | None = None) -> None:
        self._redis = redis
        self._fallback: dict[str, list[tuple[float, str]]] = {}

    def attach(self, redis: Any) -> None:
        self._redis = redis

    @staticmethod
    def _key(session_id: str) -> str:
        return f"loopguard:{session_id}"

    async def check(self, session_id: str, tool_name: str, arguments: dict[str, Any]) -> LoopCheck:
        sig = f"{tool_name}|{_args_hash(arguments)}"
        now = time.time()
        history = await self._record(session_id, sig, now)
        occurrences = sum(1 for s in history if s == sig)
        if occurrences >= THRESHOLD:
            return LoopCheck(
                loop=True,
                occurrences=occurrences,
                message=(
                    f"loop detected: tool '{tool_name}' has been called {occurrences} times with the same arguments "
                    "in the last few rounds"
                ),
                suggestion=(
                    "Stop calling this tool with these arguments. Either: (a) explain the situation to the user "
                    "and ask for clarification, (b) try a DIFFERENT tool or different arguments, or (c) summarize "
                    "what is currently known and ask the user how to proceed. Do not retry with the same payload."
                ),
            )
        return LoopCheck(loop=False, occurrences=occurrences, message=None, suggestion=None)

    async def reset(self, session_id: str) -> None:
        if self._redis is not None:
            try:
                await self._redis.delete(self._key(session_id))
            except Exception:
                pass
        self._fallback.pop(self._key(session_id), None)

    async def _record(self, session_id: str, sig: str, now: float) -> list[str]:
        key = self._key(session_id)
        if self._redis is None:
            arr = self._fallback.setdefault(key, [])
            arr.append((now, sig))
            del arr[:-WINDOW]
            return [s for _, s in arr]
        try:
            pipe = self._redis.pipeline()
            pipe.rpush(key, sig)
            pipe.ltrim(key, -WINDOW, -1)
            pipe.expire(key, KEY_TTL)
            pipe.lrange(key, 0, -1)
            results = await pipe.execute()
            history = results[-1] or []
            return [h if isinstance(h, str) else h.decode() for h in history]
        except Exception as exc:
            logger.debug("loop guard fallback: %s", exc)
            arr = self._fallback.setdefault(key, [])
            arr.append((now, sig))
            del arr[:-WINDOW]
            return [s for _, s in arr]
