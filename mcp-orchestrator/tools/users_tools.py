"""MCP tool definitions for the User service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_delete, api_get, api_post, api_put


def register(mcp: FastMCP) -> None:
    """Register all user-related tools with the MCP server."""

    @mcp.tool()
    async def users_login(email: str, password: str) -> dict[str, Any]:
        """Authenticate a user and get JWT access + refresh tokens.

        Args:
            email: User email address.
            password: User password.
        """
        return await api_post("/api/v1/auth/login", {
            "email": email, "password": password,
        })

    @mcp.tool()
    async def users_register(
        email: str,
        password: str,
        first_name: str,
        last_name: str,
        role: str,
    ) -> dict[str, Any]:
        """Register a new user (admin only).

        Args:
            email: User email address (must be unique).
            password: User password.
            first_name: User's first name.
            last_name: User's last name.
            role: User role (admin, warehouse_manager, logistics_manager, analyst, operator).
        """
        return await api_post("/api/v1/auth/register", {
            "email": email, "password": password,
            "first_name": first_name, "last_name": last_name,
            "role": role,
        })

    @mcp.tool()
    async def users_refresh_token(refresh_token: str) -> dict[str, Any]:
        """Get a new access token using a refresh token.

        Args:
            refresh_token: The refresh token from a previous login.
        """
        return await api_post("/api/v1/auth/refresh", {
            "refresh_token": refresh_token,
        })

    @mcp.tool()
    async def users_password_reset(email: str) -> dict[str, Any]:
        """Request a password reset. A reset token will be sent via email (mock adapter).

        Args:
            email: The email address of the account to reset.
        """
        return await api_post("/api/v1/auth/password-reset", {"email": email})

    @mcp.tool()
    async def users_password_reset_confirm(token: str, new_password: str) -> dict[str, Any]:
        """Confirm a password reset with the token received via email.

        Args:
            token: The password reset token.
            new_password: The new password to set.
        """
        return await api_post("/api/v1/auth/password-reset/confirm", {
            "token": token, "new_password": new_password,
        })

    @mcp.tool()
    async def users_me() -> dict[str, Any]:
        """Get the profile of the currently authenticated user."""
        return await api_get("/api/v1/users/me")

    @mcp.tool()
    async def users_update_profile(
        first_name: str,
        last_name: str,
        email: str,
    ) -> dict[str, Any]:
        """Update the profile of the currently authenticated user. Cannot change role.

        Args:
            first_name: Updated first name.
            last_name: Updated last name.
            email: Updated email address.
        """
        return await api_put("/api/v1/users/me", {
            "first_name": first_name, "last_name": last_name,
            "email": email,
        })

    @mcp.tool()
    async def users_list(
        role: str | None = None,
        email: str | None = None,
        name: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 20,
        offset: int = 0,
    ) -> dict[str, Any]:
        """List all users with optional filters (admin only).

        Args:
            role: Filter by role (admin, warehouse_manager, logistics_manager, analyst, operator).
            email: Filter by email (partial match).
            name: Filter by name (partial match).
            sort_by: Sort field (created_at, email, first_name, last_name, role).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
        """
        return await api_get("/api/v1/users", {
            "role": role, "email": email, "name": name,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def users_create(
        email: str,
        password: str,
        first_name: str,
        last_name: str,
        role: str,
    ) -> dict[str, Any]:
        """Create a new user with role assignment (admin only).

        Args:
            email: User email address (must be unique).
            password: User password.
            first_name: User's first name.
            last_name: User's last name.
            role: User role (admin, warehouse_manager, logistics_manager, analyst, operator).
        """
        return await api_post("/api/v1/users", {
            "email": email, "password": password,
            "first_name": first_name, "last_name": last_name,
            "role": role,
        })

    @mcp.tool()
    async def users_update(
        user_id: str,
        first_name: str,
        last_name: str,
        email: str,
        role: str,
    ) -> dict[str, Any]:
        """Update an existing user including role change (admin only).

        Args:
            user_id: The unique identifier of the user.
            first_name: Updated first name.
            last_name: Updated last name.
            email: Updated email address.
            role: Updated role (admin, warehouse_manager, logistics_manager, analyst, operator).
        """
        return await api_put(f"/api/v1/users/{user_id}", {
            "first_name": first_name, "last_name": last_name,
            "email": email, "role": role,
        })

    @mcp.tool()
    async def users_delete(user_id: str) -> dict[str, Any]:
        """Soft-delete a user (admin only).

        Args:
            user_id: The unique identifier of the user to delete.
        """
        return await api_delete(f"/api/v1/users/{user_id}")
