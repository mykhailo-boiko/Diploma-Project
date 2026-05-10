"""MCP tool definitions for the Logistics service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put, api_get_all


def register(mcp: FastMCP) -> None:
    """Register all logistics-related tools with the MCP server."""


    @mcp.tool()
    async def shipments_list(
        status: str | None = None,
        carrier_id: str | None = None,
        order_id: str | None = None,
        warehouse_id: str | None = None,
        date_from: str | None = None,
        date_to: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """List shipments with optional filters and pagination.

        For per-carrier on-time stats, use analytics_carriers_performance (do NOT iterate this list).
        For shipped→cancelled forensics, use analytics_quick_cancellations.
        For bulk reroute by city, use shipments_reassign_carrier (do NOT update individually).

        Args:
            status: Filter by shipment status (created, picked_up, in_transit, delivered, failed, returned).
            carrier_id: Filter by carrier ID.
            order_id: Filter by order ID.
            warehouse_id: Filter by warehouse ID.
            date_from: Filter shipments created after this date (RFC3339 format).
            date_to: Filter shipments created before this date (RFC3339 format).
            sort_by: Sort field (created_at, status, order_id).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/shipments", {
            "status": status, "carrier_id": carrier_id,
            "order_id": order_id, "warehouse_id": warehouse_id,
            "date_from": date_from, "date_to": date_to,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def shipments_get(shipment_id: str) -> dict[str, Any]:
        """Get detailed information about a specific shipment.

        Args:
            shipment_id: The unique identifier of the shipment.
        """
        return await api_get(f"/api/v1/shipments/{shipment_id}")

    @mcp.tool()
    async def shipments_create(
        order_id: str,
        warehouse_id: str,
        carrier_id: str,
        address: str,
    ) -> dict[str, Any]:
        """Create a new shipment for an order. Status starts as 'created'.

        Args:
            order_id: The order ID this shipment fulfills.
            warehouse_id: The warehouse shipping from.
            carrier_id: The carrier handling delivery.
            address: Delivery address.
        """
        return await api_post("/api/v1/shipments", {
            "order_id": order_id, "warehouse_id": warehouse_id,
            "carrier_id": carrier_id, "address": address,
        })

    @mcp.tool()
    async def shipments_update_status(shipment_id: str, status: str) -> dict[str, Any]:
        """Update the status of a shipment. Valid transitions: created->picked_up->in_transit->delivered/failed->returned.

        Args:
            shipment_id: The unique identifier of the shipment.
            status: The new status (picked_up, in_transit, delivered, failed, returned).
        """
        return await api_put(f"/api/v1/shipments/{shipment_id}/status", {"status": status})

    @mcp.tool()
    async def shipments_bulk_status(
        updates: list[dict[str, str]],
    ) -> dict[str, Any]:
        """Update the status of multiple shipments at once. Supports partial failure.

        Args:
            updates: List of status updates. Each must have: shipment_id (str), status (str).
        """
        return await api_post("/api/v1/shipments/bulk-status", {"updates": updates})


    @mcp.tool()
    async def carriers_list(
        type: str | None = None,
        is_active: str | None = None,
        name: str | None = None,
        sort_by: str | None = None,
        sort_order: str | None = None,
        limit: int = 100,
        offset: int = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        """List carriers (basic info) with optional filters and pagination.

        For per-carrier on-time rate and worst-cities breakdown, use analytics_carriers_performance.
        For bulk shipment reroute by city, use shipments_reassign_carrier.

        Args:
            type: Filter by carrier type (ground, air, sea).
            is_active: Filter by active status (true or false).
            name: Filter by carrier name (partial match).
            sort_by: Sort field (created_at, name, type, cost_per_km).
            sort_order: Sort direction (asc or desc).
            limit: Maximum number of results (default 20).
            offset: Number of results to skip (default 0).
            fetch_all: When True, automatically fetches every page and returns the full list. Use this when the user asks for "all", "everything", or otherwise wants no pagination.
        """
        return await (api_get_all if fetch_all else api_get)("/api/v1/carriers", {
            "type": type, "is_active": is_active, "name": name,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool()
    async def carriers_get(carrier_id: str) -> dict[str, Any]:
        """Get detailed information about a specific carrier.

        Args:
            carrier_id: The unique identifier of the carrier.
        """
        return await api_get(f"/api/v1/carriers/{carrier_id}")

    @mcp.tool()
    async def carriers_create(
        name: str,
        type: str,
        cost_per_km: float,
    ) -> dict[str, Any]:
        """Create a new carrier.

        Args:
            name: Carrier company name.
            type: Transport type (ground, air, sea).
            cost_per_km: Cost per kilometer in currency units.
        """
        return await api_post("/api/v1/carriers", {
            "name": name, "type": type, "cost_per_km": cost_per_km,
        })

    @mcp.tool()
    async def carriers_update(
        carrier_id: str,
        name: str,
        type: str,
        cost_per_km: float,
        is_active: bool = True,
    ) -> dict[str, Any]:
        """Update an existing carrier.

        Args:
            carrier_id: The unique identifier of the carrier.
            name: Updated carrier name.
            type: Updated transport type (ground, air, sea).
            cost_per_km: Updated cost per kilometer.
            is_active: Whether the carrier is active.
        """
        return await api_put(f"/api/v1/carriers/{carrier_id}", {
            "name": name, "type": type,
            "cost_per_km": cost_per_km, "is_active": is_active,
        })

    @mcp.tool()
    async def shipments_reassign_carrier(
        from_carrier_id: str,
        to_carrier_id: str,
        city: str | None = None,
        statuses: list[str] | None = None,
    ) -> dict[str, Any]:
        """Bulk-reassign shipments from one carrier to another, optionally filtered by
        destination city. Only shipments still in motion are touched — by default this is
        created / picked_up / in_transit. Delivered, cancelled, returned shipments are
        immutable and excluded.

        Use this when the user wants to redirect traffic away from a poorly performing
        carrier in a specific zone, e.g. 'move all pending Kharkiv shipments from
        TransContinental to SkyFreight'.

        Returns: {total, reassigned_ids, from_carrier_id, from_carrier_name, to_carrier_id,
        to_carrier_name, city}. The reassigned_ids list is the authoritative set of
        shipments actually moved — report only these to the user.

        Args:
            from_carrier_id: Carrier UUID to move shipments away from.
            to_carrier_id: Carrier UUID to assign to. Must differ from from_carrier_id.
            city: Optional destination-city filter (matched against the 3rd CSV field of
                shipment.address, ILIKE). Omit to reassign all qualifying shipments.
            statuses: Optional list of shipment statuses to touch. Defaults to
                ['created','picked_up','in_transit']. Pass an explicit list to widen or narrow.
        """
        body: dict[str, Any] = {
            "from_carrier_id": from_carrier_id,
            "to_carrier_id": to_carrier_id,
        }
        if city:
            body["city"] = city
        if statuses:
            body["statuses"] = statuses
        return await api_post("/api/v1/shipments/reassign-carrier", body)


    @mcp.tool()
    async def routes_calculate(
        origin: str,
        destination: str,
        carrier_id: str,
    ) -> dict[str, Any]:
        """Calculate a delivery route with distance, duration, and cost.

        Args:
            origin: Origin location name.
            destination: Destination location name.
            carrier_id: The carrier to use for cost calculation.
        """
        return await api_post("/api/v1/routes/calculate", {
            "origin": origin, "destination": destination,
            "carrier_id": carrier_id,
        })


    @mcp.tool()
    async def logistics_performance() -> dict[str, Any]:
        """Get delivery performance metrics: total delivered, on-time count, late count, and on-time rate."""
        return await api_get("/api/v1/logistics/performance")
