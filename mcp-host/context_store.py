
from __future__ import annotations

import json
import logging
from typing import Any

import redis.asyncio as aioredis
from google.genai import types

from config import SESSION_TTL

logger = logging.getLogger(__name__)

_HISTORY_KEY = "history:{session_id}"

class ContextStore:

    def __init__(self, redis: aioredis.Redis) -> None:
        self._redis = redis

    async def load(self, session_id: str) -> list[types.Content]:

        key = _HISTORY_KEY.format(session_id=session_id)
        data = await self._redis.get(key)
        if data is None:
            return []

        raw = data.decode() if isinstance(data, bytes) else data
        try:
            items: list[dict[str, Any]] = json.loads(raw)
        except (json.JSONDecodeError, ValueError):
            logger.warning("Corrupt history for session %s — resetting", session_id)
            return []

        return _deserialize_history(items)

    async def save(self, session_id: str, history: list[types.Content]) -> None:

        key = _HISTORY_KEY.format(session_id=session_id)
        data = json.dumps(_serialize_history(history))
        await self._redis.set(key, data, ex=SESSION_TTL)
        logger.debug("Saved history for session %s (%d turns, TTL=%ds)", session_id, len(history), SESSION_TTL)

    async def delete(self, session_id: str) -> None:

        key = _HISTORY_KEY.format(session_id=session_id)
        await self._redis.delete(key)
        logger.info("Cleared history for session %s", session_id)

def _serialize_history(history: list[types.Content]) -> list[dict[str, Any]]:

    result: list[dict[str, Any]] = []
    for content in history:
        parts: list[dict[str, Any]] = []
        for part in content.parts:
            if part.text is not None:
                parts.append({"text": part.text})
            elif part.function_call is not None:
                fc = part.function_call
                parts.append({
                    "function_call": {
                        "name": fc.name,
                        "args": dict(fc.args) if fc.args else {},
                        **({"id": fc.id} if fc.id else {}),
                    },
                })
            elif part.function_response is not None:
                fr = part.function_response
                parts.append({
                    "function_response": {
                        "name": fr.name,
                        "response": fr.response,
                        **({"id": fr.id} if fr.id else {}),
                    },
                })
        result.append({"role": content.role, "parts": parts})
    return result

def _deserialize_history(items: list[dict[str, Any]]) -> list[types.Content]:

    history: list[types.Content] = []
    for item in items:
        parts: list[types.Part] = []
        for p in item.get("parts", []):
            if "text" in p:
                parts.append(types.Part(text=p["text"]))
            elif "function_call" in p:
                fc_data = p["function_call"]
                parts.append(types.Part(function_call=types.FunctionCall(
                    name=fc_data["name"],
                    args=fc_data.get("args", {}),
                    id=fc_data.get("id"),
                )))
            elif "function_response" in p:
                fr_data = p["function_response"]
                parts.append(types.Part(function_response=types.FunctionResponse(
                    name=fr_data["name"],
                    response=fr_data.get("response", {}),
                    id=fr_data.get("id"),
                )))
        history.append(types.Content(role=item.get("role", "user"), parts=parts))
    return history
