from __future__ import annotations

from dataclasses import dataclass

import jwt

from config import JWT_ALGORITHM, JWT_SECRET

@dataclass(frozen=True)
class UserClaims:
    user_id: str
    email: str
    role: str

class AuthError(Exception):
    pass

def validate_token(token: str) -> UserClaims:
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
