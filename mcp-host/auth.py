"""JWT token validation for WebSocket handshake."""

from __future__ import annotations

from dataclasses import dataclass

import jwt

from config import JWT_ALGORITHM, JWT_SECRET


@dataclass(frozen=True)
class UserClaims:
    """Decoded JWT claims."""

    user_id: str
    email: str
    role: str


class AuthError(Exception):
    """Raised when JWT validation fails."""


def validate_token(token: str) -> UserClaims:
    """Decode and validate a JWT token, returning user claims.

    Raises AuthError on any validation failure.
    """
    try:
        payload = jwt.decode(token, JWT_SECRET, algorithms=[JWT_ALGORITHM])
        user_id = payload.get("user_id", "")
        email = payload.get("email", "")
        role = payload.get("role", "")
        if not user_id:
            raise AuthError("missing user_id in token")
        return UserClaims(user_id=user_id, email=email, role=role)
    except jwt.ExpiredSignatureError:
        raise AuthError("token expired")
    except jwt.InvalidTokenError as exc:
        raise AuthError(f"invalid token: {exc}")
