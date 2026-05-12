
from typing import Any, Literal

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put, api_get_all
from types_mcp import (
    NotificationType, PageLimit, PageOffset, SortOrder, UserRole, UUIDStr,
)

def register(mcp: FastMCP) -> None:

    @mcp.tool(description="List notifications for the CURRENT user (admin in MCP context). For per-user unread counts use notifications_unread_counts. For bulk sending use notifications_bulk. Args: type: Filter by notification type enum. sort_by: Sort field. sort_order: 'asc' or 'desc'. limit: Page size. offset: Page offset. fetch_all: Paginate through everything.")
    async def notifications_list(
        type: NotificationType | None = None,
        sort_by: Literal["created_at", "type", "status"] | None = None,
        sort_order: SortOrder | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/notifications", {
            "type": type, "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Create a notification for a user (admin only). Args: user_id: Recipient user UUID. type: Notification type enum. title: Notification title. message: Body text.")
    async def notifications_create(
        user_id: UUIDStr,
        type: NotificationType,
        title: str,
        message: str,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/notifications", {
            "user_id": user_id, "type": type,
            "title": title, "message": message,
        })

    @mcp.tool(description="Mark a notification as read. Args: notification_id: Notification UUID.")
    async def notifications_mark_read(notification_id: UUIDStr) -> dict[str, Any]:
        return await api_put(f"/api/v1/notifications/{notification_id}/read")

    @mcp.tool(description="Count of unread notifications for the current user.")
    async def notifications_unread_count() -> dict[str, Any]:
        return await api_get("/api/v1/notifications/unread-count")

    @mcp.tool(description="Unread counts per user, enriched with email/name/role (admin only). Args: role: Optional role filter.")
    async def notifications_unread_counts(role: UserRole | None = None) -> dict[str, Any]:
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

    @mcp.tool(description="Get notification preferences for current user (per-type channel toggles).")
    async def notifications_preferences_get() -> dict[str, Any]:
        return await api_get("/api/v1/notifications/preferences")

    @mcp.tool(description="Update notification preferences for a type. Args: type: Notification type enum. in_app: Enable in-app channel. email: Enable email channel. sms: Enable SMS channel.")
    async def notifications_preferences_update(
        type: NotificationType,
        in_app: bool = True,
        email: bool = True,
        sms: bool = False,
    ) -> dict[str, Any]:
        return await api_put("/api/v1/notifications/preferences", {
            "type": type, "in_app": in_app, "email": email, "sms": sms,
        })

    @mcp.tool(description="Send a notification to many users (admin only). Args: user_ids: Recipient UUIDs. type: Notification type enum. title: Title. message: Body.")
    async def notifications_bulk(
        user_ids: list[UUIDStr],
        type: NotificationType,
        title: str,
        message: str,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/notifications/bulk", {
            "user_ids": user_ids, "type": type, "title": title, "message": message,
        })
