"""MCP tool definitions for the Inventory service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_delete, api_get, api_post, api_put, api_get_all


def register(mcp: FastMCP) -> None:
    """Register all inventory-related tools with the MCP server."""


    @mcp.tool()
    async def products_list(
        sku: str | None = None,
        name: str | None = None,
        category: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """List products with optional filters and pagination.

        Args:
            sku: Filter by SKU (exact match).
            name: Filter by product name (partial match).
            category: Filter by category (exact match).
            sort_by: Sort field (created_at, name, sku, category, unit_price).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/products", {
            "sku": sku, "name": name, "category": category,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def products_get(product_id: str) -> dict[str, Any]:
        """Get detailed information about a specific product.

        Args:
            product_id: The unique identifier of the product.
        """
        return await api_get(f"/api/v1/products/{product_id}")

    @mcp.tool()
    async def products_create(
        sku: str,
        name: str,
        description: str = "",
        category: str = "",
        unit_price: float = 0.0,
    ) -> dict[str, Any]:
        """Create a new product. SKU must be unique.

        Args:
            sku: Stock Keeping Unit identifier (must be unique).
            name: Product name.
            description: Product description.
            category: Product category.
            unit_price: Price per unit.
        """
        return await api_post("/api/v1/products", {
            "sku": sku, "name": name, "description": description,
            "category": category, "unit_price": unit_price,
        })

    @mcp.tool()
    async def products_update(
        product_id: str,
        name: str,
        description: str = "",
        category: str = "",
        unit_price: float = 0.0,
    ) -> dict[str, Any]:
        """Update an existing product.

        Args:
            product_id: The unique identifier of the product.
            name: Updated product name.
            description: Updated description.
            category: Updated category.
            unit_price: Updated price per unit.
        """
        return await api_put(f"/api/v1/products/{product_id}", {
            "name": name, "description": description,
            "category": category, "unit_price": unit_price,
        })

    @mcp.tool()
    async def products_delete(product_id: str) -> dict[str, Any]:
        """Soft-delete a product. The product will no longer appear in listings.

        Args:
            product_id: The unique identifier of the product.
        """
        return await api_delete(f"/api/v1/products/{product_id}")


    @mcp.tool()
    async def warehouses_list(
        name: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """List warehouses with optional filters and pagination.

        Args:
            name: Filter by warehouse name (partial match).
            sort_by: Sort field (created_at, name).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/warehouses", {
            "name": name, "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def warehouses_get(warehouse_id: str) -> dict[str, Any]:
        """Get detailed information about a specific warehouse.

        Args:
            warehouse_id: The unique identifier of the warehouse.
        """
        return await api_get(f"/api/v1/warehouses/{warehouse_id}")

    @mcp.tool()
    async def warehouses_create(name: str, address: str = "") -> dict[str, Any]:
        """Create a new warehouse.

        Args:
            name: Warehouse name.
            address: Warehouse address.
        """
        return await api_post("/api/v1/warehouses", {"name": name, "address": address})

    @mcp.tool()
    async def warehouses_update(
        warehouse_id: str,
        name: str,
        address: str = "",
        is_active: bool = True,
    ) -> dict[str, Any]:
        """Update an existing warehouse.

        Args:
            warehouse_id: The unique identifier of the warehouse.
            name: Updated warehouse name.
            address: Updated address.
            is_active: Whether the warehouse is active.
        """
        return await api_put(f"/api/v1/warehouses/{warehouse_id}", {
            "name": name, "address": address, "is_active": is_active,
        })


    @mcp.tool()
    async def stock_list(
        product_id: str | None = None,
        warehouse_id: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """List stock levels per (product, warehouse): quantity, reserved, available, min_threshold.

        DO NOT use this to derive cross-warehouse rebalancing — use
        analytics_rebalancing_recommendations (server-side pivot + cost model).
        DO NOT use this for low-stock report — use stock_low (already filtered + joined).
        Also see: stock_movements (history), analytics_inventory_summary (window aggregates).

        Args:
            product_id: Filter by product ID.
            warehouse_id: Filter by warehouse ID.
            sort_by: Sort field (updated_at, product_id, warehouse_id, quantity, available).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/stock", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def stock_reserve(
        product_id: str,
        warehouse_id: str,
        quantity: int,
        reference: str = "",
    ) -> dict[str, Any]:
        """Reserve stock for an order. Uses atomic locking to prevent overselling.

        Args:
            product_id: The product to reserve.
            warehouse_id: The warehouse to reserve from.
            quantity: Number of units to reserve.
            reference: Reference identifier (e.g., order ID).
        """
        return await api_post("/api/v1/stock/reserve", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "quantity": quantity, "reference": reference,
        })

    @mcp.tool()
    async def stock_release(
        product_id: str,
        warehouse_id: str,
        quantity: int,
        reference: str = "",
    ) -> dict[str, Any]:
        """Release previously reserved stock.

        Args:
            product_id: The product to release.
            warehouse_id: The warehouse to release from.
            quantity: Number of units to release.
            reference: Reference identifier (e.g., order ID).
        """
        return await api_post("/api/v1/stock/release", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "quantity": quantity, "reference": reference,
        })

    @mcp.tool()
    async def stock_adjust(
        product_id: str,
        warehouse_id: str,
        quantity: int,
        type: str,
        reference: str = "",
    ) -> dict[str, Any]:
        """Adjust stock quantity (inbound, outbound, or manual adjustment).

        Args:
            product_id: The product to adjust.
            warehouse_id: The warehouse to adjust.
            quantity: Number of units to adjust (positive value).
            type: Adjustment type (inbound, outbound, adjustment).
            reference: Reference identifier.
        """
        return await api_post("/api/v1/stock/adjust", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "quantity": quantity, "type": type, "reference": reference,
        })

    @mcp.tool()
    async def stock_movements(
        stock_id: str | None = None,
        product_id: str | None = None,
        warehouse_id: str | None = None,
        type: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """Get stock movement history with optional filters.

        Args:
            stock_id: Filter by stock record ID.
            product_id: Filter by product ID.
            warehouse_id: Filter by warehouse ID.
            type: Filter by movement type (reserve, release, inbound, outbound, adjustment).
            date_from: Filter movements after this date (RFC3339 format).
            date_to: Filter movements before this date (RFC3339 format).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/stock/movements", {
            "stock_id": stock_id, "product_id": product_id,
            "warehouse_id": warehouse_id, "type": type,
            "date_from": date_from, "date_to": date_to,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def stock_low() -> dict[str, Any]:
        """Get products with stock below their min_threshold. Includes product name, SKU, warehouse name.

        For stock-out forecasting (days_to_stockout based on velocity), combine with
        orders_sales_by_product. For cross-warehouse rebalancing, use
        analytics_rebalancing_recommendations.
        """
        return await api_get("/api/v1/stock/low")

    @mcp.tool()
    async def stock_threshold_update(
        product_id: str,
        warehouse_id: str,
        threshold: int,
    ) -> dict[str, Any]:
        """Update the minimum stock threshold for a product-warehouse combination.

        Args:
            product_id: The product ID.
            warehouse_id: The warehouse ID.
            threshold: The new minimum threshold (0 = disabled).
        """
        return await api_put("/api/v1/stock/threshold", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "threshold": threshold,
        })

    @mcp.tool()
    async def inventory_report() -> dict[str, Any]:
        """Get a comprehensive inventory report with totals by warehouse and category."""
        return await api_get("/api/v1/inventory/report")
