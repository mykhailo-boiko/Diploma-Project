"""MCP tool definitions for the Analytics service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post


def register(mcp: FastMCP) -> None:
    """Register all analytics-related tools with the MCP server."""

    @mcp.tool()
    async def analytics_sales(date_from: str, date_to: str) -> dict[str, Any]:
        """Daily total sales (revenue + order count) per day for a date range.

        DO NOT use this for per-SKU velocity — use orders_sales_by_product instead.
        DO NOT use this for per-carrier metrics — use analytics_carriers_performance.
        Also see: analytics_sales_summary (totals over the range), analytics_sales_trends
        (day or week buckets), analytics_period_comparison (delta vs another period).

        Args:
            date_from: Start date (YYYY-MM-DD format, e.g. 2026-04-01).
            date_to: End date (YYYY-MM-DD format, e.g. 2026-04-15).
        """
        return await api_get("/api/v1/analytics/sales", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_sales_summary(date_from: str, date_to: str) -> dict[str, Any]:
        """Aggregated sales summary over a window: total revenue, order count, average order value.

        For per-product breakdown, use orders_sales_by_product instead.
        For period-over-period delta, use analytics_period_comparison.
        Also see: analytics_sales (daily series), analytics_sales_trends (week buckets).

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
        """Sales trends over time, bucketed by day or week.

        For totals (single number), use analytics_sales_summary.
        For per-SKU velocity, use orders_sales_by_product.
        Also see: analytics_sales (raw daily series), analytics_period_comparison.

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
        """Daily inventory snapshots (aggregated across warehouses) for a date range.

        For per-warehouse SKU view, use stock_list. For rebalancing recommendations,
        use analytics_rebalancing_recommendations. Also see: analytics_inventory_summary.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/inventory", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_inventory_summary(date_from: str, date_to: str) -> dict[str, Any]:
        """Aggregated inventory summary: total stock, reserved, available, low-stock count, turnover rate.

        For per-SKU/per-warehouse details, use stock_list (or stock_low for low-stock).
        Also see: analytics_inventory (daily series), analytics_rebalancing_recommendations.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/inventory/summary", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_logistics(date_from: str, date_to: str) -> dict[str, Any]:
        """Daily aggregated logistics metrics (all carriers combined) for a date range.

        For per-carrier on-time rate, use analytics_carriers_performance.
        For shipped-then-cancelled forensics, use analytics_quick_cancellations.
        Also see: analytics_logistics_performance (window aggregates).

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/logistics", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_logistics_performance(date_from: str, date_to: str) -> dict[str, Any]:
        """Aggregated logistics performance over a window: total shipments, delivered count, on-time rate, avg delivery hours.

        Aggregated across ALL carriers — for per-carrier breakdown use analytics_carriers_performance.
        Also see: analytics_logistics (daily series), analytics_quick_cancellations.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/logistics/performance", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_anomalies(date_from: str, date_to: str) -> dict[str, Any]:
        """Detect anomalies across four domains via rule-based thresholds:
        - sales: revenue >2σ above mean, zero-order days
        - logistics: failure rate >20%, on-time rate <80%
        - inventory: low-stock products >10% of catalog
        - business: AOV drop >2σ below recent mean

        Returns items with category (sales|inventory|logistics|business), type, severity (warning|critical),
        metric, value, threshold, date, and human-readable message.

        For drill-down of a specific anomaly day, use analytics_period_comparison and orders_list.
        Also see: analytics_quick_cancellations (carrier-handover anomalies), orders_customer_summary.

        Args:
            date_from: Start date (YYYY-MM-DD format).
            date_to: End date (YYYY-MM-DD format).
        """
        return await api_get("/api/v1/analytics/anomalies", {
            "date_from": date_from, "date_to": date_to,
        })

    @mcp.tool()
    async def analytics_what_if(
        kind: str,
        params: dict[str, Any],
    ) -> dict[str, Any]:
        """Counterfactual simulator — projects business impact of a hypothetical change.
        Use for ANY 'what if', 'simulate', 'what would happen if', 'project impact of' question.

        Supported scenarios:

            kind="carrier_drop" — what if we disable carrier X?
              params: {"carrier_id": uuid}
              Returns: platform on-time rate baseline vs projected (excluding that carrier).

            kind="capacity_increase" — what if warehouse X capacity is +N%?
              params: {"warehouse_name": str, "capacity_increase_pct": float}
              Returns: load-vs-capacity overflow index baseline vs projected.

            kind="price_change" — what if we change prices in category X by N%?
              params: {"category": str, "price_change_pct": float,
                       "elasticity": float (default -1.0),
                       "category_revenue_share": float (default 0.09 = 1/11 categories)}
              Returns: category revenue + total revenue baseline vs projected.

            kind="promo_burst" — what if demand spikes Nx for D days?
              params: {"order_multiplier": float, "duration_days": int (default 7)}
              Returns: projected revenue uplift during burst window.

        Every result includes assumptions[] and confidence_qualitative (low|medium|high).
        These are SIMPLE models — quote the assumptions list in your answer for honesty.

        Args:
            kind: One of carrier_drop, capacity_increase, price_change, promo_burst.
            params: Scenario-specific parameters (see per-kind list above).
        """
        return await api_post("/api/v1/analytics/what-if", {"kind": kind, "params": params})

    @mcp.tool()
    async def analytics_forecast(
        metric: str,
        horizon_days: int = 14,
        history_days: int = 30,
        method: str = "linear",
    ) -> dict[str, Any]:
        """Time-series forecast for a metric over a horizon, with confidence interval and
        backtest MAPE accuracy. SERVER-SIDE — do NOT compute linear extrapolation manually.

        Supported metrics: revenue, order_count, shipment_count.
        Supported methods:
            - linear: linear regression on history (good for trends)
            - rolling-avg: 7-day moving average projection (good for stationary series)
            - ets-simple: Holt's linear trend exponential smoothing (good for mild trends with noise)

        Returns: {metric, method, horizon_days, history_window_days, history[], forecast[],
                  backtest_mape, confidence_qualitative (high|medium|low), assumptions[]}.
        Each point has date, value, confidence_low, confidence_high (mean ± 1.5σ of residuals).

        ALWAYS quote method + backtest_mape + confidence label in your reply for honesty.

        Args:
            metric: One of revenue, order_count, shipment_count.
            horizon_days: How many days to forecast forward (default 14).
            history_days: Trailing window used to fit the model (default 30).
            method: linear | rolling-avg | ets-simple.
        """
        return await api_get("/api/v1/analytics/forecast", {
            "metric": metric,
            "horizon_days": horizon_days,
            "history_days": history_days,
            "method": method,
        })

    @mcp.tool()
    async def audit_query(
        actor_email: str | None = None,
        action: str | None = None,
        entity_id: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
        limit: int = 50,
    ) -> dict[str, Any]:
        """Query the audit trail of write operations performed across the platform (admin only).

        Use for ANY 'who did X', 'who changed', 'audit trail', 'show me activity',
        'find action on entity Y', 'compliance check', 'what AI did' question.

        Logged actions include (action names):
            - orders.create, orders.update_status, orders.cancel, orders.bulk_update_status
            - shipments.reassign_carrier, carriers.update
            - (more services will register their actions here over time)

        Each entry contains: actor_user_id, actor_email, actor_role, service_name, action,
        entity_type, entity_ids (array), params_snip (truncated JSON), result_status
        (success|partial|failed), success_count, failure_count, error_message, created_at.

        Args:
            actor_email: Filter to one actor (exact email).
            action: Filter to one action (exact match, e.g. "orders.bulk_update_status").
            entity_id: Filter to entries that touched this entity ID (e.g. order UUID).
            date_from: Start date (YYYY-MM-DD inclusive).
            date_to: End date (YYYY-MM-DD inclusive, expanded to end-of-day).
            limit: Max rows (default 50, max 500).
        """
        params: dict[str, Any] = {"limit": limit}
        if actor_email:
            params["actor_email"] = actor_email
        if action:
            params["action"] = action
        if entity_id:
            params["entity_id"] = entity_id
        if date_from:
            params["from"] = date_from
        if date_to:
            params["to"] = date_to
        return await api_get("/api/v1/analytics/audit-log", params)

    @mcp.tool()
    async def analytics_period_comparison(
        metric: str,
        a_from: str,
        a_to: str,
        b_from: str,
        b_to: str,
        a_label: str | None = None,
        b_label: str | None = None,
    ) -> dict[str, Any]:
        """Compare a metric between TWO arbitrary date windows. Returns both values, absolute and
        percent delta, direction, and a qualitative significance label.

        Use for ANY "X vs Y" question: this month vs last, this Q vs same Q last year, week-over-week,
        before-and-after a campaign, etc. The model should ALWAYS compute concrete YYYY-MM-DD dates
        for both windows itself — do not pass quarter/month names.

        Supported metrics:
            - revenue: sum of total_amount for non-cancelled/non-returned orders in window
            - order_count: count of non-cancelled/non-returned orders
            - aov: average order value
            - cancellation_rate: % of orders in window that ended up cancelled
            - on_time_rate: % of delivered shipments within 168h SLA
            - shipment_count: count of shipments created in window
            - low_stock_count: current count (window-independent; both periods will return the same)

        Returns:
            {metric, period_a:{label,from,to,value}, period_b:{label,from,to,value},
             absolute_delta, percent_change, direction: up|down|flat,
             significance: major|minor|noise}
        Significance thresholds: |percent_change| >= 15% = major, >= 5% = minor, otherwise noise.

        Args:
            metric: One of the supported metrics above.
            a_from, a_to: Baseline (period A) window in YYYY-MM-DD inclusive.
            b_from, b_to: Comparison (period B) window in YYYY-MM-DD inclusive.
            a_label, b_label: Optional human-readable labels (e.g. "Q1 2026" / "Q4 2025").
        """
        params: dict[str, Any] = {
            "metric": metric,
            "a_from": a_from, "a_to": a_to,
            "b_from": b_from, "b_to": b_to,
        }
        if a_label:
            params["a_label"] = a_label
        if b_label:
            params["b_label"] = b_label
        return await api_get("/api/v1/analytics/period-comparison", params)

    @mcp.tool()
    async def customers_profile_360(
        customer_name: str,
        recent_n: int = 5,
        top_categories_n: int = 5,
    ) -> dict[str, Any]:
        """Full executive-grade profile of a single customer in ONE call: lifetime aggregates,
        churn risk, top spending categories, recent orders, status mix, behaviour metrics.

        Use this for ANY 'everything about customer X', 'full profile of', 'customer overview',
        'customer dossier', 'how valuable is this customer', 'is this customer at risk' question.
        DO NOT compose this from orders_list + orders_get + orders_customer_summary — one call is faster
        and atomic.

        Result fields:
            - lifetime_value, order_count, avg_order_value, first_order_date, last_order_date
            - days_since_last_order, median_inter_order_days
            - churn_risk_score (0..1, computed as 1 - exp(-days_since_last/(2 * median_inter_order)))
            - is_new_customer_90_days (bool)
            - status_breakdown: map of order status -> count for this customer
            - top_categories: top-N spending categories (revenue + units)
            - recent_orders: last-N order headers

        Cancelled and returned orders are excluded from lifetime aggregates (but included in
        status_breakdown so you see the full mix).

        Args:
            customer_name: Exact customer name (case-sensitive). Use orders_search if you have
                only partial info.
            recent_n: How many most-recent orders to include (default 5).
            top_categories_n: How many top spending categories to include (default 5).
        """
        return await api_get("/api/v1/analytics/customers/profile-360", {
            "customer_name": customer_name,
            "recent_n": recent_n,
            "top_categories_n": top_categories_n,
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
        """High-level reorder advice (single summary row): how many SKUs are below threshold and
        a recommended quantity for the worst.

        For per-SKU per-warehouse rebalancing with explicit cost/ROI model, use
        analytics_rebalancing_recommendations — it is significantly more actionable.
        For per-SKU velocity, use orders_sales_by_product.

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
        """Generate a multi-section analytics report combining several sub-metrics.

        Prefer specific tools for narrow questions (analytics_sales_summary, analytics_carriers_performance,
        analytics_anomalies). Use analytics_report only when the user asks for a 'full report' or
        wants all sections together.

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
