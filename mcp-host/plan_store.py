
from __future__ import annotations

import logging
from typing import Any

import redis.asyncio as aioredis

from models import ExecutionPlan

logger = logging.getLogger(__name__)

_PLAN_KEY = "plan:{session_id}:{plan_id}"
_SESSION_INDEX = "plans:{session_id}"
_TTL_SECONDS = 86400

class PlanStore:

    def __init__(self, redis: aioredis.Redis) -> None:
        self._redis = redis

    async def save(self, plan: ExecutionPlan) -> None:

        key = _PLAN_KEY.format(session_id=plan.session_id, plan_id=plan.id)
        index_key = _SESSION_INDEX.format(session_id=plan.session_id)

        data = plan.model_dump_json()

        pipe = self._redis.pipeline()
        pipe.set(key, data, ex=_TTL_SECONDS)
        pipe.zadd(index_key, {plan.id: plan.created_at.timestamp()})
        pipe.expire(index_key, _TTL_SECONDS)
        await pipe.execute()

        logger.debug("Saved plan %s for session %s", plan.id, plan.session_id)

    async def get(self, session_id: str, plan_id: str) -> ExecutionPlan | None:

        key = _PLAN_KEY.format(session_id=session_id, plan_id=plan_id)
        data = await self._redis.get(key)
        if data is None:
            return None
        return ExecutionPlan.model_validate_json(data)

    async def list_by_session(self, session_id: str) -> list[dict[str, Any]]:

        index_key = _SESSION_INDEX.format(session_id=session_id)
        plan_ids = await self._redis.zrevrange(index_key, 0, -1)

        plans: list[dict[str, Any]] = []
        for pid_bytes in plan_ids:
            pid = pid_bytes.decode() if isinstance(pid_bytes, bytes) else pid_bytes
            key = _PLAN_KEY.format(session_id=session_id, plan_id=pid)
            data = await self._redis.get(key)
            if data is None:
                continue
            plan = ExecutionPlan.model_validate_json(data)
            plans.append({
                "id": plan.id,
                "intent": plan.intent,
                "status": plan.status.value,
                "steps_count": len(plan.steps),
                "created_at": plan.created_at.isoformat(),
                "finished_at": plan.finished_at.isoformat() if plan.finished_at else None,
            })
        return plans

    async def health(self) -> bool:

        try:
            return await self._redis.ping()
        except Exception:
            return False
