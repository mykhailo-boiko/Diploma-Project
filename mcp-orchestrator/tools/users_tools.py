
from typing import Any, Literal

from mcp.server.fastmcp import FastMCP

from http_client import api_delete, api_get, api_post, api_put, api_get_all
from types_mcp import (
    EmailAddr, PageLimit, PageOffset, SortOrder, UserRole, UUIDStr,
)

def register(mcp: FastMCP) -> None:

    @mcp.tool(description="Authenticate a user and get JWT access + refresh tokens. Args: email: User email address. password: User password.")
    async def users_login(email: EmailAddr, password: str) -> dict[str, Any]:
        return await api_post("/api/v1/auth/login", {"email": email, "password": password})

    @mcp.tool(description="Register a new user (admin only). Args: email: Unique email address. password: Password (min 8 chars recommended). first_name: First name. last_name: Last name. role: Role enum.")
    async def users_register(
        email: EmailAddr,
        password: str,
        first_name: str,
        last_name: str,
        role: UserRole,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/auth/register", {
            "email": email, "password": password,
            "first_name": first_name, "last_name": last_name, "role": role,
        })

    @mcp.tool(description="Get a new access token using a refresh token. Args: refresh_token: Refresh token from previous login.")
    async def users_refresh_token(refresh_token: str) -> dict[str, Any]:
        return await api_post("/api/v1/auth/refresh", {"refresh_token": refresh_token})

    @mcp.tool(description="Request a password reset email. Args: email: Account email.")
    async def users_password_reset(email: EmailAddr) -> dict[str, Any]:
        return await api_post("/api/v1/auth/password-reset", {"email": email})

    @mcp.tool(description="Confirm a password reset with the token received via email. Args: token: Password reset token. new_password: New password.")
    async def users_password_reset_confirm(token: str, new_password: str) -> dict[str, Any]:
        return await api_post("/api/v1/auth/password-reset/confirm", {
            "token": token, "new_password": new_password,
        })

    @mcp.tool(description="Get the profile of the currently authenticated user.")
    async def users_me() -> dict[str, Any]:
        return await api_get("/api/v1/users/me")

    @mcp.tool(description="Update profile of the currently authenticated user. Cannot change role. Args: first_name: Updated first name. last_name: Updated last name. email: Updated email.")
    async def users_update_profile(
        first_name: str,
        last_name: str,
        email: EmailAddr,
    ) -> dict[str, Any]:
        return await api_put("/api/v1/users/me", {
            "first_name": first_name, "last_name": last_name, "email": email,
        })

    @mcp.tool(description="List users (admin only). Args: role: Role enum filter. email: Partial-match email. name: Partial-match name. sort_by: Sort field. sort_order: 'asc' or 'desc'. limit: Page size. offset: Page offset. fetch_all: Paginate through everything.")
    async def users_list(
        role: UserRole | None = None,
        email: str | None = None,
        name: str | None = None,
        sort_by: Literal["created_at", "email", "first_name", "last_name", "role"] | None = None,
        sort_order: SortOrder | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/users", {
            "role": role, "email": email, "name": name,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Create a new user with role (admin only). Args: email: Unique email. password: Password. first_name: First name. last_name: Last name. role: Role enum.")
    async def users_create(
        email: EmailAddr,
        password: str,
        first_name: str,
        last_name: str,
        role: UserRole,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/users", {
            "email": email, "password": password,
            "first_name": first_name, "last_name": last_name, "role": role,
        })

    @mcp.tool(description="Update a user including role (admin only). Args: user_id: User UUID. first_name: Updated first name. last_name: Updated last name. email: Updated email. role: Updated role enum.")
    async def users_update(
        user_id: UUIDStr,
        first_name: str,
        last_name: str,
        email: EmailAddr,
        role: UserRole,
    ) -> dict[str, Any]:
        return await api_put(f"/api/v1/users/{user_id}", {
            "first_name": first_name, "last_name": last_name,
            "email": email, "role": role,
        })

    @mcp.tool(description="Soft-delete a user (admin only). Args: user_id: User UUID.")
    async def users_delete(user_id: UUIDStr) -> dict[str, Any]:
        return await api_delete(f"/api/v1/users/{user_id}")
