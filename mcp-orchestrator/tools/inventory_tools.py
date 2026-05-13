
from typing import Any, Literal

from mcp.server.fastmcp import FastMCP

from http_client import api_delete, api_get, api_post, api_put, api_get_all
from types_mcp import (
    ISODateTime, Money, NonNegativeInt, PageLimit, PageOffset,
    PositiveInt, ProductCategory, SortOrder, StockMovementType, UUIDStr,
)

def register(mcp: FastMCP) -> None:

    @mcp.tool(description="List products with filters and pagination. Args: sku: Exact SKU filter. name: Partial-match name filter. category: Product category enum. sort_by: Sort field. sort_order: 'asc' or 'desc'. limit: Page size. offset: Page offset. fetch_all: Paginate through everything.")
    async def products_list(
        sku: str | None = None,
        name: str | None = None,
        category: ProductCategory | None = None,
        sort_by: Literal["created_at", "name", "sku", "category", "unit_price"] | None = None,
        sort_order: SortOrder | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/products", {
            "sku": sku, "name": name, "category": category,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Get a single product. Args: product_id: Product UUID.")
    async def products_get(product_id: UUIDStr) -> dict[str, Any]:
        return await api_get(f"/api/v1/products/{product_id}")

    @mcp.tool(description="Create a new product. SKU must be unique. Args: sku: Stock Keeping Unit identifier (unique). name: Product name. description: Product description. category: Product category enum. unit_price: Price per unit (>= 0).")
    async def products_create(
        sku: str,
        name: str,
        description: str = "",
        category: ProductCategory = "Other",
        unit_price: Money = 0.0,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/products", {
            "sku": sku, "name": name, "description": description,
            "category": category, "unit_price": unit_price,
        })

    @mcp.tool(description="Update an existing product. Args: product_id: Product UUID. name: Updated name. description: Updated description. category: Updated category enum. unit_price: Updated price.")
    async def products_update(
        product_id: UUIDStr,
        name: str,
        description: str = "",
        category: ProductCategory = "Other",
        unit_price: Money = 0.0,
    ) -> dict[str, Any]:
        return await api_put(f"/api/v1/products/{product_id}", {
            "name": name, "description": description,
            "category": category, "unit_price": unit_price,
        })

    @mcp.tool(description="Soft-delete a product. Args: product_id: Product UUID.")
    async def products_delete(product_id: UUIDStr) -> dict[str, Any]:
        return await api_delete(f"/api/v1/products/{product_id}")

    @mcp.tool(description="List warehouses. Args: name: Partial-match name filter. sort_by: Sort field. sort_order: 'asc' or 'desc'. limit: Page size. offset: Page offset. fetch_all: Paginate through everything.")
    async def warehouses_list(
        name: str | None = None,
        sort_by: Literal["created_at", "name"] | None = None,
        sort_order: SortOrder | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/warehouses", {
            "name": name, "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Get a single warehouse. Args: warehouse_id: Warehouse UUID.")
    async def warehouses_get(warehouse_id: UUIDStr) -> dict[str, Any]:
        return await api_get(f"/api/v1/warehouses/{warehouse_id}")

    @mcp.tool(description="Create a new warehouse. Args: name: Warehouse name. address: Address (free-form).")
    async def warehouses_create(name: str, address: str = "") -> dict[str, Any]:
        return await api_post("/api/v1/warehouses", {"name": name, "address": address})

    @mcp.tool(description="Update a warehouse. Args: warehouse_id: Warehouse UUID. name: Updated name. address: Updated address. is_active: Active flag.")
    async def warehouses_update(
        warehouse_id: UUIDStr,
        name: str,
        address: str = "",
        is_active: bool = True,
    ) -> dict[str, Any]:
        return await api_put(f"/api/v1/warehouses/{warehouse_id}", {
            "name": name, "address": address, "is_active": is_active,
        })

    @mcp.tool(description="List stock levels per (product, warehouse). For cross-warehouse rebalancing use analytics_rebalancing_recommendations. For low-stock use stock_low. Args: product_id: Product UUID filter. warehouse_id: Warehouse UUID filter. sort_by: Sort field. sort_order: 'asc' or 'desc'. limit: Page size. offset: Page offset. fetch_all: Paginate through everything.")
    async def stock_list(
        product_id: UUIDStr | None = None,
        warehouse_id: UUIDStr | None = None,
        sort_by: Literal["updated_at", "product_id", "warehouse_id", "quantity", "available"] | None = None,
        sort_order: SortOrder | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/stock", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Reserve stock atomically (prevents overselling). Args: product_id: Product UUID. warehouse_id: Warehouse UUID. quantity: Units to reserve (> 0). reference: Reference identifier (e.g. order UUID).")
    async def stock_reserve(
        product_id: UUIDStr,
        warehouse_id: UUIDStr,
        quantity: PositiveInt,
        reference: str = "",
    ) -> dict[str, Any]:
        return await api_post("/api/v1/stock/reserve", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "quantity": quantity, "reference": reference,
        })

    @mcp.tool(description="Release previously reserved stock. Args: product_id: Product UUID. warehouse_id: Warehouse UUID. quantity: Units to release (> 0). reference: Reference identifier.")
    async def stock_release(
        product_id: UUIDStr,
        warehouse_id: UUIDStr,
        quantity: PositiveInt,
        reference: str = "",
    ) -> dict[str, Any]:
        return await api_post("/api/v1/stock/release", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "quantity": quantity, "reference": reference,
        })

    @mcp.tool(description="Adjust stock quantity. Quantity is ALWAYS positive — `type` determines direction. - 'inbound' (restock/supply): increases stock. - 'outbound' (sale/damage): decreases stock; fails if insufficient available stock. - 'adjustment' (correction): increases stock; use for inventory corrections. Args: product_id: Product UUID. warehouse_id: Warehouse UUID. quantity: Positive integer (> 0). type: Movement type enum. reference: Reference identifier.")
    async def stock_adjust(
        product_id: UUIDStr,
        warehouse_id: UUIDStr,
        quantity: PositiveInt,
        type: StockMovementType,
        reference: str = "",
    ) -> dict[str, Any]:
        return await api_post("/api/v1/stock/adjust", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "quantity": quantity, "type": type, "reference": reference,
        })

    @mcp.tool(description="Stock movement history. Args: stock_id: Stock record UUID filter. product_id: Product UUID filter. warehouse_id: Warehouse UUID filter. type: Movement type filter. date_from: RFC3339 created-after. date_to: RFC3339 created-before. limit: Page size. offset: Page offset. fetch_all: Paginate through everything.")
    async def stock_movements(
        stock_id: UUIDStr | None = None,
        product_id: UUIDStr | None = None,
        warehouse_id: UUIDStr | None = None,
        type: Literal["reserve", "release", "inbound", "outbound", "adjustment"] | None = None,
        date_from: ISODateTime | None = None,
        date_to: ISODateTime | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/stock/movements", {
            "stock_id": stock_id, "product_id": product_id,
            "warehouse_id": warehouse_id, "type": type,
            "date_from": date_from, "date_to": date_to,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Products with stock below min_threshold. Includes product name, SKU, warehouse name. For stock-out forecasting combine with orders_sales_by_product. For cross-warehouse rebalancing use analytics_rebalancing_recommendations.")
    async def stock_low() -> dict[str, Any]:
        return await api_get("/api/v1/stock/low")

    @mcp.tool(description="Update minimum stock threshold for a (product, warehouse) pair. Args: product_id: Product UUID. warehouse_id: Warehouse UUID. min_threshold: New min threshold (0 = disabled).")
    async def stock_threshold_update(
        product_id: UUIDStr,
        warehouse_id: UUIDStr,
        min_threshold: NonNegativeInt,
    ) -> dict[str, Any]:
        return await api_put("/api/v1/stock/threshold", {
            "product_id": product_id, "warehouse_id": warehouse_id,
            "min_threshold": min_threshold,
        })

    @mcp.tool(description="Comprehensive inventory report with totals by warehouse and category.")
    async def inventory_report() -> dict[str, Any]:
        return await api_get("/api/v1/inventory/report")
