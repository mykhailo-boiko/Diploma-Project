"""MCP tool definitions for the Analytics service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post


def register(mcp: FastMCP) -> None:
    """Register all analytics-related tools with the MCP server."""

    @mcp.tool()
    async def analytics_sales(date_from: str, date_to: str) -> dict[str, Any]:
        """Get daily sales data for a date range.

        Args:
            date_from: Start date (YYYY-MM-DD format, e.g. 2026-04-01).
            date_to: End date (YYYY-MM-DD format, e.g. 2026-04-15).
        """
        return await api_get("/api/v1/analytics/sales", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_sales_summary(date_from: str, date_to: str) -> dict[str, Any]:
        """Get aggregated sales summary: total revenue, order count, and average order value.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/sales/summary", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_sales_trends(
        date_from: str,
        date_to: str,
        granularity: str = "day",
    ) -> dict[str, Any]:
        """Get sales trends over time, aggregated by day or week.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
            granularity: Time bucket size (day or week, default: day).
        """
        return await api_get("/api/v1/analytics/sales/trends", {
            "date_from": date_from, "date_to": date_to,
            "granularity": granularity,
        })

    @mcp.tool()
    async def analytics_inventory(date_from: str, date_to: str) -> dict[str, Any]:
        """Get daily inventory snapshots for a date range.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/inventory", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_inventory_summary(date_from: str, date_to: str) -> dict[str, Any]:
        """Get aggregated inventory summary: total stock, reserved, available, low-stock count, and turnover rate.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/inventory/summary", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_logistics(date_from: str, date_to: str) -> dict[str, Any]:
        """Get daily logistics metrics for a date range.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/logistics", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_logistics_performance(date_from: str, date_to: str) -> dict[str, Any]:
        """Get logistics performance analysis: shipment counts, delivery rate, on-time rate, average delivery time.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/logistics/performance", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_anomalies(date_from: str, date_to: str) -> dict[str, Any]:
        """Detect anomalies in sales, inventory, and logistics data using rule-based thresholds.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/anomalies", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_carriers_performance(
        date_from: str,
        date_to: str,
        sla_hours: int = 168,
        worst_cities: int = 5,
    ) -> dict[str, Any]:
        """Per-carrier delivery performance for a date range, with a breakdown of the worst
        destination cities for each carrier.

        Use this for any 'on-time rate per carrier', 'which carrier is worst/best', 'carrier
        SLA violation', 'carrier scorecard', or 'carrier × city anomaly' question. There is
        no other way to get per-carrier on-time stats — analytics_logistics_performance is
        aggregated across all carriers and won't help.

        Each result row has: carrier_id, carrier_name, is_active, total_shipments, delivered,
        on_time, late, cancelled, returned, on_time_rate (0..1), avg_delivery_hours, and
        worst_cities[] (top N cities for that carrier sorted by late share desc). Rows are
        sorted ascending by on_time_rate so the WORST performer is first and the BEST is last.

        Args:
            date_from: Window start inclusive (YYYY-MM-DD).
            date_to: Window end inclusive (YYYY-MM-DD).
            sla_hours: SLA window in hours used to classify on-time vs late deliveries.
                Default 168 (7 days, industry ground-freight standard).
            worst_cities: Top N cities to include in each carrier's worst_cities list. Default 5.
        """
        return await api_get("/api/v1/analytics/carriers-performance", {
            "date_from": date_from, "date_to": date_to,
            "sla_hours": sla_hours, "worst_cities": worst_cities,
        })

    @mcp.tool()
    async def analytics_rebalancing_recommendations(
        overstock_multiplier: float = 3.0,
        holding_daily_rate: float = 0.0005,
        holding_horizon_days: int = 30,
        transfer_base_fee: float = 50.0,
        transfer_per_unit: float = 1.5,
        include_unprofitable: bool = False,
        limit: int = 50,
    ) -> dict[str, Any]:
        """Cross-warehouse stock rebalancing recommendations with realistic economic model.

        Identifies SKUs that are simultaneously overstocked at one warehouse (donor) and below
        threshold at a DIFFERENT warehouse (acceptor), and proposes internal transfers ranked
        by net economic benefit and ROI. Server-side pivots stock × warehouse and ENFORCES that
        donor and acceptor are different warehouses (a common LLM mistake when handling raw stock_list).

        Use this for any 'rebalance', 'overstock vs understock', 'internal transfer',
        'inventory redistribution', or 'where should I move excess stock' question. DO NOT
        try to derive this from raw stock_list — the per-SKU pivot, cost model and ROI
        ranking are non-trivial and easy to get wrong.

        Cost model (defaults are realistic B2B internal-transfer assumptions, override if needed):
            - holding_savings = transfer_qty × unit_price × holding_daily_rate × holding_horizon_days
              (default 0.0005/day = ~1.5%/month carrying cost over a 30-day horizon)
            - transfer_cost = transfer_base_fee + transfer_qty × transfer_per_unit
              (default $50 dispatch fee + $1.50 per unit handling)
            - net_benefit = holding_savings − transfer_cost
            - roi_pct = net_benefit / transfer_cost × 100

        Each row is one (SKU, donor_warehouse, acceptor_warehouse) recommendation, picking the
        BEST donor per (SKU, acceptor) pair. Rows are sorted by net_benefit descending.

        Args:
            overstock_multiplier: A warehouse is donor if quantity > min_threshold × this. Default 3.
            holding_daily_rate: Carrying cost as fraction of unit_price per day. Default 0.0005.
            holding_horizon_days: How long the saved holding cost is amortized over. Default 30.
            transfer_base_fee: Fixed transfer cost per recommendation. Default $50.
            transfer_per_unit: Variable transfer cost per unit moved. Default $1.50.
            include_unprofitable: If True, also return rows with negative net_benefit (default False).
            limit: Max recommendations to return. Default 50.
        """
        return await api_get("/api/v1/analytics/rebalancing", {
            "overstock_multiplier": overstock_multiplier,
            "holding_daily_rate": holding_daily_rate,
            "holding_horizon_days": holding_horizon_days,
            "transfer_base_fee": transfer_base_fee,
            "transfer_per_unit": transfer_per_unit,
            "include_unprofitable": "true" if include_unprofitable else "false",
            "limit": limit,
        })

    @mcp.tool()
    async def analytics_quick_cancellations(
        date_from: str,
        date_to: str,
        max_minutes: int = 60,
    ) -> dict[str, Any]:
        """Forensic report: orders cancelled within max_minutes after their shipment was created
        (i.e. shortly after status became 'shipped'), grouped by carrier × destination city.

        Use this for any 'cancelled right after shipped' / 'quick cancel' / 'last-minute cancellation'
        / 'shipping-handover failure' question. The endpoint joins orders × shipments × carriers
        across schemas server-side — there is no other way to answer this without an N+1 client loop.

        Each result row contains: carrier_name, city, count, avg_minutes_between, min_minutes_between,
        max_minutes_between, lost_revenue, sample_order_ids, sample_cancel_reasons. Rows are sorted
        by count descending. The biggest count × lost_revenue row is the anomaly hotspot.

        Args:
            date_from: Window start inclusive (YYYY-MM-DD).
            date_to: Window end inclusive (YYYY-MM-DD).
            max_minutes: Cutoff for 'quick' cancellation (default 60). Set 30 for tighter, 120 for looser.
        """
        return await api_get("/api/v1/analytics/quick-cancellations", {
            "date_from": date_from, "date_to": date_to,
            "max_minutes": max_minutes,
        })

    @mcp.tool()
    async def analytics_optimization(date_from: str, date_to: str) -> dict[str, Any]:
        """Get reorder optimization recommendations based on demand analysis and safety stock calculations.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/optimization", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_report(
        report_type: str,
        date_from: str,
        date_to: str,
    ) -> dict[str, Any]:
        """Generate a custom analytics report.

        Args:
            report_type: Type of report to generate (sales, inventory, logistics, full).
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_post("/api/v1/analytics/report", {
            "report_type": report_type,
            "date_from": date_from,
            "date_to": date_to,
        })
