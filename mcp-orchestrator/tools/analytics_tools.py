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
