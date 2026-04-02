"""MCP tool definitions for the Notification service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put


def register(mcp: FastMCP) -> None:
    """Register all notification-related tools with the MCP server."""

    @mcp.tool()
    async def notifications_list(
        type: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 20,
        offset: int = 0,
    ) -> dict[str, Any]:
        """List notifications for the current user with optional filters.

        Args:
            type: Filter by notification type (order_created, order_updated, order_cancelled, low_stock, stock_changed, shipment_created, shipment_updated, system).
            sort_by: Sort field (created_at, type, status).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
        """
        return await api_get("/api/v1/notifications", {
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
