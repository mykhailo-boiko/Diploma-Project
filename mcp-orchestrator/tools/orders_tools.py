"""MCP tool definitions for the Order service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put, api_get_all


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
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """List orders (header-level: id, customer, status, total, dates). Does NOT include
        line_items — use orders_get for that.

        DO NOT iterate orders_get over this list for per-SKU velocity — use orders_sales_by_product.
        For customer cohort/behaviour, use orders_customer_summary.
        For bulk status change, use orders_bulk_update_status.
        Also see: orders_search (free-text), orders_stats (status histogram).

        Args:
            status: Filter by order status (pending, confirmed, processing, shipped, delivered, completed, cancelled, returned).
            date_from: Filter orders created after this date (RFC3339 format, e.g. 2026-01-01T00:00:00Z).
            date_to: Filter orders created before this date (RFC3339 format).
            customer_name: Filter by customer name (partial match).
            sort_by: Sort field (created_at, total_amount, status, customer_name).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results to return (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/orders", {
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
        """Get a SINGLE order with its line items (product_id, name, quantity, unit_price, subtotal).

        DO NOT loop this over a list of orders to compute aggregates — use orders_sales_by_product
        for per-SKU velocity or orders_customer_summary for per-customer rollup.

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
        """Update the status of a single order. Valid transitions:
        pending -> confirmed -> processing -> {shipped | confirmed (rollback for review)},
        shipped -> {delivered | returned | cancelled}, delivered -> completed.

        Args:
            order_id: The unique identifier of the order.
            status: The new status (confirmed, processing, shipped, delivered, completed,
                cancelled, returned).
        """
        return await api_put(f"/api/v1/orders/{order_id}/status", {"status": status})

    @mcp.tool()
    async def orders_bulk_update_status(
        order_ids: list[str],
        status: str,
        note: str | None = None,
        dry_run: bool = False,
    ) -> dict[str, Any]:
        """Update status for many orders in ONE server-side call. Validates each transition,
        skips invalid ones, and returns a per-order success/failure report. Use this whenever
        you need to flip more than 2-3 orders — DO NOT loop orders_update_status, you will
        run out of tool-call rounds.

        Returns: {total, updated_ids, successes:[{order_id, old_status, new_status}],
                  failures:[{order_id, old_status, new_status, error}]}.

        The 'updated_ids' field is the authoritative list of orders that actually changed
        status — use it (and ONLY it) when reporting back to the user. Never claim an order
        was updated unless its ID appears in updated_ids.

        Args:
            order_ids: List of order UUIDs (max 500 per call).
            status: Target status (confirmed, processing, shipped, delivered, completed,
                cancelled, returned). Each order is validated against the state machine;
                invalid transitions land in 'failures', not 'successes'.
            note: Optional service comment stored on each updated order (e.g.
                "stalled — needs review"). Persisted to cancel_reason field.
            dry_run: When True, validates every transition and reports what WOULD happen
                (same response shape, with dry_run=true) without writing anything. ALWAYS
                use dry_run=true first when affecting more than 5-10 orders, then ask the
                user to confirm before re-running with dry_run=false.
        """
        body: dict[str, Any] = {"order_ids": order_ids, "status": status, "dry_run": dry_run}
        if note:
            body["note"] = note
        return await api_post("/api/v1/orders/bulk-status", body)

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
        """Order statistics: total count, total revenue, and breakdown by status.

        For per-SKU sales, use orders_sales_by_product. For per-customer, use orders_customer_summary.
        For window-bounded analytics, use analytics_sales_summary.
        """
        return await api_get("/api/v1/orders/stats")

    @mcp.tool()
    async def orders_customer_summary(
        date_from: str | None = None,
        date_to: str | None = None,
        only_new: bool = False,
        sort_by: str | None = None,
        sort_order: str = "desc",
        limit: int = 0,
    ) -> dict[str, Any]:
        """Per-customer aggregate: lifetime first/last order date, total orders, total revenue,
        avg order value, plus orders/revenue within an optional window and a 'new_in_window' flag.

        Use this for any question about customer cohorts, behaviour, or value:
            * 'new vs returning customers' (filter rows by new_in_window field after calling)
            * 'top N customers by lifetime / window revenue'
            * 'churn risk' / 'inactive customers' (sort by last_order, filter old)
            * 'first-time buyers in this period'
            * customer-cohort breakdown of revenue spikes
        Cancelled and returned orders are excluded.

        When date_from + date_to are passed: the result is restricted to customers with at least
        one order in that window, and orders_in_window / revenue_in_window / new_in_window are
        populated. new_in_window=True means the customer's first-ever order falls inside the window.

        Args:
            date_from: Optional window start (YYYY-MM-DD or RFC3339).
            date_to: Optional window end (YYYY-MM-DD or RFC3339).
            only_new: If True, return only customers whose first-ever order is in the window
                (date_from/date_to required).
            sort_by: revenue (lifetime) | revenue_in_window | orders | last_order | first_order.
                Defaults to revenue_in_window when window is set, otherwise revenue.
            sort_order: asc or desc (default desc).
            limit: Cap on rows returned. 0 = no limit.
        """
        params: dict[str, Any] = {"sort_order": sort_order}
        if date_from:
            params["date_from"] = date_from
        if date_to:
            params["date_to"] = date_to
        if only_new:
            params["only_new"] = "true"
        if sort_by:
            params["sort_by"] = sort_by
        if limit > 0:
            params["limit"] = limit
        return await api_get("/api/v1/orders/customers", params)

    @mcp.tool()
    async def orders_sales_by_product(
        date_from: str,
        date_to: str,
        statuses: str | None = None,
        limit: int = 0,
    ) -> dict[str, Any]:
        """Get aggregated sales per product (SKU) for a date range. Returns units_sold,
        revenue, order_count, and daily_demand for each product.

        Use this for: per-SKU velocity, top selling products, demand calculation, stock-out
        forecasting, and any 'which product sells the most / least' question. This is the
        ONLY way to get per-product sales data — orders_list does NOT include line items.

        Args:
            date_from: Start date inclusive (YYYY-MM-DD).
            date_to: End date inclusive (YYYY-MM-DD).
            statuses: Optional comma-separated order statuses to count. Defaults to
                'confirmed,processing,shipped,delivered,completed' (excludes cancelled/returned).
            limit: Optional cap on rows returned (sorted by revenue desc). 0 means no cap.
        """
        params: dict[str, Any] = {"date_from": date_from, "date_to": date_to}
        if statuses:
            params["statuses"] = statuses
        if limit > 0:
            params["limit"] = limit
        return await api_get("/api/v1/orders/sales-by-product", params)
