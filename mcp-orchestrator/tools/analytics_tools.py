
from typing import Any, Literal

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post
from types_mcp import (
    AnalyticsMetric, AnomalyCategory, ForecastMethod,
    ISODate, NonNegativeInt, PositiveInt, UUIDStr,
)

def register(mcp: FastMCP) -> None:

    @mcp.tool(description="Daily total sales (revenue + order count) per day for a date range. DO NOT use this for per-SKU velocity — use orders_sales_by_product instead. DO NOT use this for per-carrier metrics — use analytics_carriers_performance. Also see: analytics_sales_summary, analytics_sales_trends, analytics_period_comparison. Args: date_from: Start date in 'YYYY-MM-DD' format (inclusive), e.g. '2026-04-01'. date_to: End date in 'YYYY-MM-DD' format (inclusive), e.g. '2026-04-15'.")
    async def analytics_sales(date_from: ISODate, date_to: ISODate) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/sales", {"date_from": date_from, "date_to": date_to})

    @mcp.tool(description="Aggregated sales totals over a window: revenue, order count, AOV. For period-vs-period delta, use analytics_period_comparison. Also see: analytics_sales (daily series), analytics_sales_trends (buckets). Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'.")
    async def analytics_sales_summary(date_from: ISODate, date_to: ISODate) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/sales/summary", {"date_from": date_from, "date_to": date_to})

    @mcp.tool(description="Sales trends over time, bucketed by 'day' or 'week'. Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'. granularity: Bucket size, 'day' or 'week'.")
    async def analytics_sales_trends(
        date_from: ISODate,
        date_to: ISODate,
        granularity: Literal["day", "week"] = "day",
    ) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/sales/trends", {
            "date_from": date_from, "date_to": date_to, "granularity": granularity,
        })

    @mcp.tool(description="Daily inventory snapshots aggregated across warehouses. For per-warehouse SKU view use stock_list. For transfers — analytics_rebalancing_recommendations. Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'.")
    async def analytics_inventory(date_from: ISODate, date_to: ISODate) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/inventory", {"date_from": date_from, "date_to": date_to})

    @mcp.tool(description="Aggregated inventory totals: stock, reserved, available, low-stock count, turnover. Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'.")
    async def analytics_inventory_summary(date_from: ISODate, date_to: ISODate) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/inventory/summary", {"date_from": date_from, "date_to": date_to})

    @mcp.tool(description="Daily aggregated logistics metrics (all carriers combined). For per-carrier on-time rate use analytics_carriers_performance. For shipped-then-cancelled forensics — analytics_quick_cancellations. Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'.")
    async def analytics_logistics(date_from: ISODate, date_to: ISODate) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/logistics", {"date_from": date_from, "date_to": date_to})

    @mcp.tool(description="Aggregated logistics performance: total shipments, delivered count, on-time rate, avg delivery hours. Aggregated across ALL carriers — for per-carrier breakdown use analytics_carriers_performance. Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'.")
    async def analytics_logistics_performance(date_from: ISODate, date_to: ISODate) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/logistics/performance", {"date_from": date_from, "date_to": date_to})

    @mcp.tool(description="Detect anomalies across four domains (sales, logistics, inventory, business). Returns items with category, type, severity (warning|critical), metric, value, threshold, date, and human-readable message. Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'. category: Filter to one domain ('sales','logistics','inventory','business','all').")
    async def analytics_anomalies(
        date_from: ISODate,
        date_to: ISODate,
        category: AnomalyCategory = "all",
    ) -> dict[str, Any]:
        params: dict[str, Any] = {"date_from": date_from, "date_to": date_to}
        if category and category != "all":
            params["category"] = category
        return await api_get("/api/v1/analytics/anomalies", params)

    @mcp.tool(description="Counterfactual simulator — projects business impact of a hypothetical change. Supported scenarios: - carrier_drop: params {carrier_id: UUID} - capacity_increase: params {warehouse_name: str, capacity_increase_pct: float} - price_change: params {category: str, price_change_pct: float, elasticity?: float, category_revenue_share?: float} - promo_burst: params {order_multiplier: float, duration_days?: int} Args: kind: Scenario kind (see list). params: Scenario-specific parameters dict.")
    async def analytics_what_if(
        kind: Literal["carrier_drop", "capacity_increase", "price_change", "promo_burst"],
        params: dict[str, Any],
    ) -> dict[str, Any]:
        return await api_post("/api/v1/analytics/what-if", {"kind": kind, "params": params})

    @mcp.tool(description="Time-series forecast for a metric over a horizon, with confidence interval and MAPE backtest. Args: metric: 'revenue', 'order_count', or 'shipment_count'. horizon_days: How many days to forecast forward (1..365). history_days: Trailing window used to fit the model (1..365). method: 'linear', 'rolling-avg', or 'ets-simple'.")
    async def analytics_forecast(
        metric: Literal["revenue", "order_count", "shipment_count"],
        horizon_days: PositiveInt = 14,
        history_days: PositiveInt = 30,
        method: ForecastMethod = "linear",
    ) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/forecast", {
            "metric": metric, "horizon_days": horizon_days,
            "history_days": history_days, "method": method,
        })

    @mcp.tool(description="Query the audit trail of write operations (admin only). Each entry contains: actor_user_id, actor_email, actor_role, service_name, action, entity_type, entity_ids[], params_snip, result_status, success_count, failure_count, error_message, created_at. Args: actor_email: Filter to one actor (exact email). action: Filter to one action (e.g. 'orders.bulk_update_status'). entity_id: UUID of an order/shipment/etc that the action touched. date_from: Start date 'YYYY-MM-DD' (inclusive). date_to: End date 'YYYY-MM-DD' (inclusive, expanded to end-of-day). limit: Max rows (1..500).")
    async def audit_query(
        actor_email: str | None = None,
        action: str | None = None,
        entity_id: UUIDStr | None = None,
        date_from: ISODate | None = None,
        date_to: ISODate | None = None,
        limit: PositiveInt = 50,
    ) -> dict[str, Any]:
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

    @mcp.tool(description="Compare a metric between TWO date windows. Returns both values, absolute and percent delta, direction, and significance label. Significance: |percent_change| >= 15% major, >= 5% minor, else noise. Args: metric: One of supported metrics (revenue, order_count, aov, cancellation_rate, on_time_rate, shipment_count, low_stock_count). a_from, a_to: Period A window (baseline) in 'YYYY-MM-DD' (inclusive). b_from, b_to: Period B window (comparison) in 'YYYY-MM-DD' (inclusive). a_label, b_label: Optional human-readable labels (e.g. 'Q1 2026' / 'Q4 2025').")
    async def analytics_period_comparison(
        metric: AnalyticsMetric,
        a_from: ISODate,
        a_to: ISODate,
        b_from: ISODate,
        b_to: ISODate,
        a_label: str | None = None,
        b_label: str | None = None,
    ) -> dict[str, Any]:
        params: dict[str, Any] = {
            "metric": metric, "a_from": a_from, "a_to": a_to,
            "b_from": b_from, "b_to": b_to,
        }
        if a_label:
            params["a_label"] = a_label
        if b_label:
            params["b_label"] = b_label
        return await api_get("/api/v1/analytics/period-comparison", params)

    @mcp.tool(description="Full executive-grade profile of a single customer in ONE call: lifetime aggregates, churn risk, top categories, recent orders, status mix. Args: customer_name: Exact customer name (case-sensitive). Use orders_search to disambiguate. recent_n: How many recent orders to include (1..50). top_categories_n: How many top categories (1..20).")
    async def customers_profile_360(
        customer_name: str,
        recent_n: PositiveInt = 5,
        top_categories_n: PositiveInt = 5,
    ) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/customers/profile-360", {
            "customer_name": customer_name,
            "recent_n": recent_n,
            "top_categories_n": top_categories_n,
        })

    @mcp.tool(description="Per-carrier delivery performance for a date range, with worst destination cities per carrier. Args: date_from: Window start 'YYYY-MM-DD' (inclusive). date_to: Window end 'YYYY-MM-DD' (inclusive). sla_hours: SLA threshold in hours (default 168 = 7 days). worst_cities: Top N cities per carrier (default 5).")
    async def analytics_carriers_performance(
        date_from: ISODate,
        date_to: ISODate,
        sla_hours: PositiveInt = 168,
        worst_cities: PositiveInt = 5,
    ) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/carriers-performance", {
            "date_from": date_from, "date_to": date_to,
            "sla_hours": sla_hours, "worst_cities": worst_cities,
        })

    @mcp.tool(description="Cross-warehouse stock rebalancing recommendations with economic model. Cost model: holding_savings = qty × unit_price × holding_daily_rate × holding_horizon_days. transfer_cost = transfer_base_fee + qty × transfer_per_unit. net_benefit = holding_savings - transfer_cost; roi_pct = net_benefit / transfer_cost × 100. Args: overstock_multiplier: Threshold for donor (qty > min_threshold × this). Default 3. holding_daily_rate: Carrying cost as fraction of unit_price/day. Default 0.0005. holding_horizon_days: Amortization horizon (days). Default 30. transfer_base_fee: Fixed cost per recommendation. Default 50.0. transfer_per_unit: Variable cost per unit moved. Default 1.5. include_unprofitable: ...")
    async def analytics_rebalancing_recommendations(
        overstock_multiplier: float = 3.0,
        holding_daily_rate: float = 0.0005,
        holding_horizon_days: PositiveInt = 30,
        transfer_base_fee: float = 50.0,
        transfer_per_unit: float = 1.5,
        include_unprofitable: bool = False,
        limit: PositiveInt = 50,
    ) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/rebalancing", {
            "overstock_multiplier": overstock_multiplier,
            "holding_daily_rate": holding_daily_rate,
            "holding_horizon_days": holding_horizon_days,
            "transfer_base_fee": transfer_base_fee,
            "transfer_per_unit": transfer_per_unit,
            "include_unprofitable": include_unprofitable,
            "limit": limit,
        })

    @mcp.tool(description="Forensic: orders cancelled within max_minutes after shipment created, grouped by carrier × city. Args: date_from: Window start 'YYYY-MM-DD'. date_to: Window end 'YYYY-MM-DD'. max_minutes: Cutoff for 'quick' cancellation (default 60).")
    async def analytics_quick_cancellations(
        date_from: ISODate,
        date_to: ISODate,
        max_minutes: PositiveInt = 60,
    ) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/quick-cancellations", {
            "date_from": date_from, "date_to": date_to, "max_minutes": max_minutes,
        })

    @mcp.tool(description="High-level reorder advice — single summary row with SKUs below threshold + recommended qty. For per-SKU rebalancing with cost model use analytics_rebalancing_recommendations. Args: date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'.")
    async def analytics_optimization(date_from: ISODate, date_to: ISODate) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/optimization", {"date_from": date_from, "date_to": date_to})

    @mcp.tool(description="Return the full audit trail for a single entity (order, shipment, product, etc.) — admin only. Returns all `audit.action_log` rows whose entity_ids array contains `entity_id`, ordered by created_at DESC, plus a list of unique trace_ids touching this entity. Use this for ANY 'who did X on order Y', 'reconstruct what happened with shipment Z', 'sequence of actions on this entity', 'why was status changed' question. Args: entity_id: UUID of the entity to trace. limit: Max events (1..1000, default 200).")
    async def audit_trace_by_entity(
        entity_id: UUIDStr,
        limit: PositiveInt = 200,
    ) -> dict[str, Any]:
        return await api_get("/api/v1/analytics/trace/by-entity", {
            "entity_id": entity_id, "limit": limit,
        })

    @mcp.tool(description="Generate multi-section analytics report. For narrow questions prefer specific tools (analytics_sales_summary, etc). Args: report_type: 'sales' | 'inventory' | 'logistics' | 'full'. date_from: Start date 'YYYY-MM-DD'. date_to: End date 'YYYY-MM-DD'.")
    async def analytics_report(
        report_type: Literal["sales", "inventory", "logistics", "full"],
        date_from: ISODate,
        date_to: ISODate,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/analytics/report", {
            "report_type": report_type, "date_from": date_from, "date_to": date_to,
        })
