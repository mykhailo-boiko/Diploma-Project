"""Tests for plan REST endpoints."""

from contextlib import asynccontextmanager
from unittest.mock import AsyncMock

import pytest
from fastapi.testclient import TestClient

import main
from main import app
from models import ExecutionPlan
from plan_store import PlanStore


@pytest.fixture
def mock_plan_store():
    store = AsyncMock(spec=PlanStore)
    store.health.return_value = True
    return store


@pytest.fixture
def client(mock_plan_store):
    @asynccontextmanager
    async def test_lifespan(app):
        yield

    app.router.lifespan_context = test_lifespan
    original = main._plan_store
    main._plan_store = mock_plan_store
    yield TestClient(app)
    main._plan_store = original


class TestListPlans:
    def test_list_plans_returns_plans(self, client, mock_plan_store):
        mock_plan_store.list_by_session.return_value = [
            {
                "id": "p1", "intent": "test", "status": "completed",
                "steps_count": 2, "created_at": "2026-01-01T00:00:00", "finished_at": None,
            },
        ]
        resp = client.get("/api/v1/mcp/plans/session1")
        assert resp.status_code == 200
        data = resp.json()
        assert "plans" in data
        assert len(data["plans"]) == 1
        assert data["plans"][0]["id"] == "p1"

    def test_list_plans_empty(self, client, mock_plan_store):
        mock_plan_store.list_by_session.return_value = []
        resp = client.get("/api/v1/mcp/plans/session1")
        assert resp.status_code == 200
        assert resp.json()["plans"] == []


class TestGetPlan:
    def test_get_existing_plan(self, client, mock_plan_store):
        plan = ExecutionPlan(session_id="s1", intent="list orders")
        step = plan.add_step(tool="orders_list", params={"limit": 10})
        plan.start_step(step)
        plan.complete_step(step, {"orders": []})
        plan.finalize()
        mock_plan_store.get.return_value = plan

        resp = client.get(f"/api/v1/mcp/plans/s1/{plan.id}")
        assert resp.status_code == 200
        data = resp.json()
        assert data["id"] == plan.id
        assert data["session_id"] == "s1"
        assert data["intent"] == "list orders"
        assert len(data["steps"]) == 1
        assert data["steps"][0]["tool"] == "orders_list"
        assert data["steps"][0]["status"] == "success"
        assert data["status"] == "completed"

    def test_get_missing_plan(self, client, mock_plan_store):
        mock_plan_store.get.return_value = None
        resp = client.get("/api/v1/mcp/plans/s1/nonexistent")
        assert resp.status_code == 404
        assert "not found" in resp.json()["error"]


class TestHealthWithRedis:
    def test_health_includes_redis(self, client, mock_plan_store):
        resp = client.get("/health")
        assert resp.status_code == 200
        data = resp.json()
        assert "redis" in data
        assert data["redis"] == "ok"
