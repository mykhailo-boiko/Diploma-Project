"""MCP tool definitions for the Notification service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put, api_get_all


def register(mcp: FastMCP) -> None:
    """Register all notification-related tools with the MCP server."""

    @mcp.tool()
    async def notifications_list(
        type: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """List notifications for the current user with optional filters.

        Args:
            type: Filter by notification type (order_created, order_updated, order_cancelled, low_stock, stock_changed, shipment_created, shipment_updated, system).
            sort_by: Sort field (created_at, type, status).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/notifications", {
            "type": type, "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def notifications_create(
        user_id: str,
        type: str,
        title: str,
        message: str,
    ) -> dict[str, Any]:
        """Create a notification for a user (admin only).

        Args:
            user_id: The recipient user ID.
            type: Notification type (order_created, order_updated, order_cancelled, low_stock, stock_changed, shipment_created, shipment_updated, system).
            title: Notification title.
            message: Notification message body.
        """
        return await api_post("/api/v1/notifications", {
            "user_id": user_id, "type": type,
            "title": title, "message": message,
        })

    @mcp.tool()
    async def notifications_mark_read(notification_id: str) -> dict[str, Any]:
        """Mark a notification as read.

        Args:
            notification_id: The unique identifier of the notification.
        """
        return await api_put(f"/api/v1/notifications/{notification_id}/read")

    @mcp.tool()
    async def notifications_unread_count() -> dict[str, Any]:
        """Get the count of unread notifications for the current user."""
        return await api_get("/api/v1/notifications/unread-count")

    @mcp.tool()
    async def notifications_unread_counts(role: str | None = None) -> dict[str, Any]:
        """Get unread notification counts for every user, enriched with email, name and role (admin only).

        Use this when the user asks which users / managers / staff have the most unread
        notifications, or wants a leaderboard of unread counts. Returns rows sorted by
        unread_count descending. Users with zero unread are excluded.

        Args:
            role: Optional role filter (admin, warehouse_manager, logistics_manager, analyst, operator).
                When set, only users with this role are returned.
        """
        counts_resp = await api_get("/api/v1/notifications/admin/unread-counts")
        users_resp = await api_get_all("/api/v1/users", {"role": role} if role else {})

        users_payload = users_resp.get("data") if isinstance(users_resp, dict) else users_resp
        if isinstance(users_payload, dict):
            users_payload = users_payload.get("data", [])
        users_by_id = {u["id"]: u for u in (users_payload or []) if isinstance(u, dict) and u.get("id")}

        counts_payload = counts_resp.get("data") if isinstance(counts_resp, dict) else counts_resp
        rows = []
        for c in (counts_payload or []):
            uid = c.get("user_id")
            unread = c.get("unread_count", 0)
            user = users_by_id.get(uid)
            if role and not user:
                continue
            rows.append({
                "user_id": uid,
                "email": user.get("email") if user else None,
                "name": (
                    f"{user.get('first_name','').strip()} {user.get('last_name','').strip()}".strip()
                    if user else None
                ),
                "role": user.get("role") if user else None,
                "unread_count": unread,
            })

        rows.sort(key=lambda r: r["unread_count"], reverse=True)
        return {"data": rows}

    @mcp.tool()
    async def notifications_preferences_get() -> dict[str, Any]:
        """Get notification preferences for the current user (per notification type channel toggles)."""
        return await api_get("/api/v1/notifications/preferences")

    @mcp.tool()
    async def notifications_preferences_update(
        type: str,
        in_app: bool = True,
        email: bool = True,
        sms: bool = False,
    ) -> dict[str, Any]:
        """Update notification preferences for a specific notification type.

        Args:
            type: Notification type to configure (order_created, order_updated, order_cancelled, low_stock, stock_changed, shipment_created, shipment_updated, system).
            in_app: Enable in-app notifications (default: true).
            email: Enable email notifications (default: true).
            sms: Enable SMS notifications (default: false).
        """
        return await api_put("/api/v1/notifications/preferences", {
            "type": type, "in_app": in_app,
            "email": email, "sms": sms,
        })

    @mcp.tool()
    async def notifications_bulk(
        user_ids: list[str],
        type: str,
        title: str,
        message: str,
    ) -> dict[str, Any]:
        """Send a notification to multiple users at once (admin only). Returns success/failure counts.

        Args:
            user_ids: List of recipient user IDs.
            type: Notification type.
            title: Notification title.
            message: Notification message body.
        """
        return await api_post("/api/v1/notifications/bulk", {
            "user_ids": user_ids, "type": type,
            "title": title, "message": message,
        })
