
from typing import Any, Literal

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put, api_get_all
from types_mcp import (
    ISODate, ISODateTime, NonNegativeInt, OrderStatus, PageLimit, PageOffset,
    Money, PositiveInt, UUIDStr,
)

def register(mcp: FastMCP) -> None:

    @mcp.tool(description="List orders (header-level). For line items use orders_get. DO NOT iterate orders_get over this list for per-SKU velocity — use orders_sales_by_product. For customer cohort/behaviour, use orders_customer_summary. For bulk status change, use orders_bulk_update_status. Args: status: Filter by order status enum. date_from: Filter orders created after this RFC3339 datetime. date_to: Filter orders created before this RFC3339 datetime. customer_name: Partial-match customer name filter. sort_by: Field to sort by. sort_order: 'asc' or 'desc'. limit: Page size (1..1000). offset: Page offset. fetch_all: If True, paginate through everything (max 5000 rows).")
    async def orders_list(
        status: OrderStatus | None = None,
        date_from: ISODateTime | None = None,
        date_to: ISODateTime | None = None,
        customer_name: str | None = None,
        sort_by: Literal["created_at", "total_amount", "status", "customer_name"] | None = None,
        sort_order: Literal["asc", "desc"] | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/orders", {
            "status": status, "date_from": date_from, "date_to": date_to,
            "customer_name": customer_name,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Get a SINGLE order with its line items (product_id, name, quantity, unit_price, subtotal). DO NOT loop this over a list of orders to compute aggregates — use orders_sales_by_product for per-SKU velocity or orders_customer_summary for per-customer rollup. Args: order_id: Order UUID.")
    async def orders_get(order_id: UUIDStr) -> dict[str, Any]:
        return await api_get(f"/api/v1/orders/{order_id}")

    @mcp.tool(description="Create a new order with line items. The order starts in 'pending' status. Args: customer_name: Full customer name (Latin transliteration), e.g. 'Yuliia Morozenko'. items: List of items, each: {product_id: UUID, name: str, quantity: int>0, unit_price: float>0}.")
    async def orders_create(
        customer_name: str,
        items: list[dict[str, Any]],
    ) -> dict[str, Any]:
        return await api_post("/api/v1/orders", {
            "customer_name": customer_name, "items": items,
        })

    @mcp.tool(description="Update the status of a single order. Valid transitions: pending→confirmed→processing→shipped→delivered→completed. Any non-terminal status can transition to cancelled. Args: order_id: Order UUID. status: Target status enum.")
    async def orders_update_status(order_id: UUIDStr, status: OrderStatus) -> dict[str, Any]:
        return await api_put(f"/api/v1/orders/{order_id}/status", {"status": status})

    @mcp.tool(description="Update status for many orders in ONE call. Validates each transition, returns per-order report. Returns: {total, updated_ids, successes:[...], failures:[...]}. Use updated_ids as the authoritative success list. ALWAYS dry_run=true first when >5 orders. Args: order_ids: List of order UUIDs (max 500). status: Target status enum. note: Optional service comment. dry_run: If True, validates without writing.")
    async def orders_bulk_update_status(
        order_ids: list[UUIDStr],
        status: OrderStatus,
        note: str | None = None,
        dry_run: bool = False,
    ) -> dict[str, Any]:
        body: dict[str, Any] = {"order_ids": order_ids, "status": status, "dry_run": dry_run}
        if note:
            body["note"] = note
        return await api_post("/api/v1/orders/bulk-status", body)

    @mcp.tool(description="Cancel an order with a reason. Args: order_id: Order UUID. reason: Non-empty cancellation reason for audit log.")
    async def orders_cancel(order_id: UUIDStr, reason: str) -> dict[str, Any]:
        return await api_post(f"/api/v1/orders/{order_id}/cancel", {"reason": reason})

    @mcp.tool(description="Search orders by customer name or order ID. Minimum 2 characters. Args: query: Search query string.")
    async def orders_search(query: str) -> dict[str, Any]:
        return await api_get("/api/v1/orders/search", {"q": query})

    @mcp.tool(description="Order statistics: total count, total revenue, breakdown by status. For per-SKU sales use orders_sales_by_product. For per-customer — orders_customer_summary. For window-bounded analytics — analytics_sales_summary.")
    async def orders_stats() -> dict[str, Any]:
        return await api_get("/api/v1/orders/stats")

    @mcp.tool(description="Per-customer aggregate: lifetime + windowed metrics. Use for cohorts, top-N, churn risk, first-time buyers. Args: date_from: Optional window start 'YYYY-MM-DD'. date_to: Optional window end 'YYYY-MM-DD'. only_new: Filter to customers whose first-ever order is in window. sort_by: Sort field; defaults to 'revenue_in_window' when window set, else 'revenue'. sort_order: 'asc' or 'desc'. limit: Max rows (0 = unlimited).")
    async def orders_customer_summary(
        date_from: ISODate | None = None,
        date_to: ISODate | None = None,
        only_new: bool = False,
        sort_by: Literal["revenue", "revenue_in_window", "orders", "last_order", "first_order"] | None = None,
        sort_order: Literal["asc", "desc"] = "desc",
        limit: NonNegativeInt = 0,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {"sort_order": sort_order}
        if date_from:
            params["date_from"] = date_from
        if date_to:
            params["date_to"] = date_to
        if only_new:
            params["only_new"] = True
        if sort_by:
            params["sort_by"] = sort_by
        if limit > 0:
            params["limit"] = limit
        return await api_get("/api/v1/orders/customers", params)

    @mcp.tool(description="Aggregated sales per product (SKU) for a date range: units_sold, revenue, order_count, daily_demand. Use for per-SKU velocity / top-sellers / demand forecasting. Args: date_from: Start date inclusive 'YYYY-MM-DD'. date_to: End date inclusive 'YYYY-MM-DD'. statuses: Comma-separated statuses to include (default 'confirmed,processing,shipped,delivered,completed'). limit: Cap on rows (0 = unlimited).")
    async def orders_sales_by_product(
        date_from: ISODate,
        date_to: ISODate,
        statuses: str | None = None,
        limit: NonNegativeInt = 0,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {"date_from": date_from, "date_to": date_to}
        if statuses:
            params["statuses"] = statuses
        if limit > 0:
            params["limit"] = limit
        return await api_get("/api/v1/orders/sales-by-product", params)
