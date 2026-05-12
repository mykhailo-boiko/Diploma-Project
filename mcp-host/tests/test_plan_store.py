
from unittest.mock import AsyncMock, MagicMock

import pytest

from models import ExecutionPlan
from plan_store import PlanStore

@pytest.fixture
def mock_redis():

    redis = AsyncMock()
    pipe = MagicMock()
    pipe.set = MagicMock(return_value=pipe)
    pipe.zadd = MagicMock(return_value=pipe)
    pipe.expire = MagicMock(return_value=pipe)
    pipe.execute = AsyncMock()
    redis.pipeline = MagicMock(return_value=pipe)
    return redis

@pytest.fixture
def store(mock_redis):
    return PlanStore(mock_redis)

class TestPlanStoreSave:
    async def test_save_creates_key_and_index(self, store, mock_redis):
        plan = ExecutionPlan(session_id="s1", intent="test")
        await store.save(plan)

        pipe = mock_redis.pipeline.return_value
        pipe.set.assert_called_once()
        pipe.zadd.assert_called_once()
        pipe.expire.assert_called_once()
        pipe.execute.assert_awaited_once()

class TestPlanStoreGet:
    async def test_get_existing_plan(self, store, mock_redis):
        plan = ExecutionPlan(session_id="s1", intent="test")
        mock_redis.get.return_value = plan.model_dump_json().encode()

        result = await store.get("s1", plan.id)
        assert result is not None
        assert result.id == plan.id
        assert result.intent == "test"

    async def test_get_missing_plan(self, store, mock_redis):
        mock_redis.get.return_value = None
        result = await store.get("s1", "nonexistent")
        assert result is None

class TestPlanStoreList:
    async def test_list_by_session(self, store, mock_redis):
        plan1 = ExecutionPlan(session_id="s1", intent="first")
        plan2 = ExecutionPlan(session_id="s1", intent="second")
        plan2.finalize()

        mock_redis.zrevrange.return_value = [plan2.id.encode(), plan1.id.encode()]

        async def get_side_effect(key):
            if plan1.id in key:
                return plan1.model_dump_json().encode()
            if plan2.id in key:
                return plan2.model_dump_json().encode()
            return None

        mock_redis.get.side_effect = get_side_effect

        plans = await store.list_by_session("s1")
        assert len(plans) == 2
        assert plans[0]["intent"] == "second"
        assert plans[1]["intent"] == "first"
        assert "id" in plans[0]
        assert "status" in plans[0]
        assert "steps_count" in plans[0]
        assert "created_at" in plans[0]

    async def test_list_empty_session(self, store, mock_redis):
        mock_redis.zrevrange.return_value = []
        plans = await store.list_by_session("empty")
        assert plans == []

class TestPlanStoreHealth:
    async def test_health_ok(self, store, mock_redis):
        mock_redis.ping.return_value = True
        assert await store.health() is True

    async def test_health_fail(self, store, mock_redis):
        mock_redis.ping.side_effect = ConnectionError("refused")
        assert await store.health() is False
