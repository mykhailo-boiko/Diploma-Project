"""Tests for JWT authentication module."""

import time

import jwt as pyjwt
import pytest

from auth import AuthError, UserClaims, validate_token
from config import JWT_ALGORITHM, JWT_SECRET


def _make_token(payload: dict, secret: str = JWT_SECRET) -> str:
    return pyjwt.encode(payload, secret, algorithm=JWT_ALGORITHM)


class TestValidateToken:
    def test_valid_token(self):
        token = _make_token({
            "user_id": "u1",
            "email": "test@example.com",
            "role": "admin",
            "exp": int(time.time()) + 3600,
        })
        claims = validate_token(token)
        assert isinstance(claims, UserClaims)
        assert claims.user_id == "u1"
        assert claims.email == "test@example.com"
        assert claims.role == "admin"

    def test_expired_token(self):
        token = _make_token({
            "user_id": "u1",
            "email": "test@example.com",
            "role": "admin",
            "exp": int(time.time()) - 100,
        })
        with pytest.raises(AuthError, match="token expired"):
            validate_token(token)

    def test_invalid_secret(self):
        token = _make_token(
            {"user_id": "u1", "email": "a@b.com", "role": "user", "exp": int(time.time()) + 3600},
            secret="wrong-secret",
        )
        with pytest.raises(AuthError, match="invalid token"):
            validate_token(token)

    def test_missing_user_id(self):
        token = _make_token({
            "email": "a@b.com",
            "role": "user",
            "exp": int(time.time()) + 3600,
        })
        with pytest.raises(AuthError, match="missing user_id"):
            validate_token(token)

    def test_malformed_token(self):
        with pytest.raises(AuthError, match="invalid token"):
            validate_token("not.a.real.token")

    def test_empty_token(self):
        with pytest.raises(AuthError, match="invalid token"):
            validate_token("")
