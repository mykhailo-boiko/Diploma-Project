"""MCP tool definitions for the Order service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put


def register(mcp: FastMCP) -> None:
    """Register all order-related tools with the MCP server."""

    @mcp.tool()
    async def orders_list(
        status: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
        customer_name: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 20,
        offset: int = 0,
    ) -> dict[str, Any]:
        """List orders with optional filters and pagination.

        Args:
            status: Filter by order status (pending, confirmed, processing, shipped, delivered, completed, cancelled, returned).
            date_from: Filter orders created after this date (RFC3339 format, e.g. 2026-01-01T00:00:00Z).
            date_to: Filter orders created before this date (RFC3339 format).
            customer_name: Filter by customer name (partial match).
            sort_by: Sort field (created_at, total_amount, status, customer_name).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results to return (default 20).
            offset: Number of results to skip (default 0).
        """
        return await api_get("/api/v1/orders", {
            "status": status,
            "date_from": date_from,
            "date_to": date_to,
            "customer_name": customer_name,
            "sort_by": sort_by,
            "sort_order": sort_order,
            "limit": limit,
            "offset": offset,
        })

    @mcp.tool()
    async def orders_get(order_id: str) -> dict[str, Any]:
        """Get detailed information about a specific order including its line items.

        Args:
            order_id: The unique identifier of the order.
        """
        return await api_get(f"/api/v1/orders/{order_id}")

    @mcp.tool()
    async def orders_create(
        customer_name: str,
        items: list[dict[str, Any]],
    ) -> dict[str, Any]:
        """Create a new order with line items. The order starts in 'pending' status.

        Args:
            customer_name: Name of the customer placing the order.
            items: List of order items. Each item must have: product_id (str), name (str), quantity (int), unit_price (float).
        """
        return await api_post("/api/v1/orders", {
            "customer_name": customer_name,
            "items": items,
        })

    @mcp.tool()
    async def orders_update_status(order_id: str, status: str) -> dict[str, Any]:
        """Update the status of an order. Valid transitions: pending->confirmed->processing->shipped->delivered->completed.

        Args:
            order_id: The unique identifier of the order.
            status: The new status (confirmed, processing, shipped, delivered, completed).
        """
        return await api_put(f"/api/v1/orders/{order_id}/status", {"status": status})

    @mcp.tool()
    async def orders_cancel(order_id: str, reason: str) -> dict[str, Any]:
        """Cancel an order with a reason. Any order can be cancelled regardless of current status.

        Args:
            order_id: The unique identifier of the order.
            reason: The reason for cancellation.
        """
        return await api_post(f"/api/v1/orders/{order_id}/cancel", {"reason": reason})

    @mcp.tool()
    async def orders_search(query: str) -> dict[str, Any]:
        """Search orders by customer name or order ID. Minimum 2 characters required.

        Args:
            query: Search query string (matches customer_name and order ID).
        """
        return await api_get("/api/v1/orders/search", {"q": query})

    @mcp.tool()
    async def orders_stats() -> dict[str, Any]:
        """Get order statistics: total count, total revenue, and breakdown by status."""
        return await api_get("/api/v1/orders/stats")
