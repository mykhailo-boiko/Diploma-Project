"""Tests for the main FastAPI application."""

import time

import jwt as pyjwt
import pytest

from config import JWT_ALGORITHM, JWT_SECRET
from main import _parse_message, app


def _make_token(user_id: str = "u1", email: str = "test@test.com", role: str = "admin") -> str:
    return pyjwt.encode(
        {"user_id": user_id, "email": email, "role": role, "exp": int(time.time()) + 3600},
        JWT_SECRET,
        algorithm=JWT_ALGORITHM,
    )


class TestParseMessage:
    def test_valid_json(self):
        result = _parse_message('{"message": "hello"}')
        assert result == {"message": "hello"}

    def test_invalid_json(self):
        assert _parse_message("not json") is None

    def test_non_dict_json(self):
        assert _parse_message("[1, 2, 3]") is None

    def test_empty_string(self):
        assert _parse_message("") is None

    def test_nested_json(self):
        result = _parse_message('{"message": "hello", "extra": {"key": "val"}}')
        assert result is not None
        assert result["message"] == "hello"


class TestHealthEndpoint:
    @pytest.fixture
    def client(self):
        from contextlib import asynccontextmanager

        from fastapi.testclient import TestClient

        @asynccontextmanager
        async def test_lifespan(app):
            yield

        app.router.lifespan_context = test_lifespan
        return TestClient(app)

    def test_health_returns_ok(self, client):
        resp = client.get("/health")
        assert resp.status_code == 200
        data = resp.json()
        assert "status" in data
        assert "tools" in data


class TestWebSocketAuth:
    @pytest.fixture
    def client(self):
        from contextlib import asynccontextmanager

        from fastapi.testclient import TestClient

        @asynccontextmanager
        async def test_lifespan(app):
            yield

        app.router.lifespan_context = test_lifespan
        return TestClient(app)

    def test_missing_token_closes(self, client):
        with pytest.raises(Exception):
            with client.websocket_connect("/ws/chat"):
                pass

    def test_invalid_token_closes(self, client):
        with pytest.raises(Exception):
            with client.websocket_connect("/ws/chat?token=bad"):
                pass

    def test_expired_token_closes(self, client):
        token = pyjwt.encode(
            {"user_id": "u1", "email": "a@b.com", "role": "admin", "exp": int(time.time()) - 100},
            JWT_SECRET,
            algorithm=JWT_ALGORITHM,
        )
        with pytest.raises(Exception):
            with client.websocket_connect(f"/ws/chat?token={token}"):
                pass
