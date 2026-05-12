"""MCP tool definitions for the Logistics service."""

from typing import Any

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put, api_patch, api_get_all


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
    async def shipments_tracking(tracking_number: str) -> dict[str, Any]:
        """Full postal tracking record for a shipment by its human-readable tracking number
        (e.g. CO-2026-K7H2P9). Returns the timeline of all events, current location, recipient and
        sender details, delivery attempts and ETA.

        Use for ANY 'where is my package', 'tracking info', 'shipment status by tracking number',
        'show me the timeline for X' question.

        Response: {shipment, events[], delivery_attempts[]}.

        Args:
            tracking_number: Human-readable code from shipment.tracking_number (NOT the UUID).
        """
        return await api_get(f"/api/v1/tracking/{tracking_number}")

    @mcp.tool()
    async def shipments_timeline(shipment_id: str) -> dict[str, Any]:
        """Full timeline of a shipment by internal UUID — events + delivery attempts + current shipment row.

        Prefer shipments_tracking when you have the tracking_number; use this when only the UUID is available.

        Args:
            shipment_id: Internal UUID of the shipment.
        """
        return await api_get(f"/api/v1/shipments/{shipment_id}/timeline")

    @mcp.tool()
    async def shipments_update_recipient(
        shipment_id: str,
        full_name: str | None = None,
        phone: str | None = None,
        email: str | None = None,
        company: str | None = None,
        street: str | None = None,
        city: str | None = None,
        region: str | None = None,
        postcode: str | None = None,
        country: str | None = None,
        delivery_notes: str | None = None,
    ) -> dict[str, Any]:
        """Partial update of shipment recipient (PATCH semantics — only provided fields change).
        Triggers a 'recipient_updated' event + audit log entry. Allowed until status=delivered.

        For destination-address changes after the shipment left the origin warehouse, prefer
        shipments_redirect instead (it also recomputes ETA).

        Args:
            shipment_id: Shipment UUID.
            full_name, phone (E.164), email, company, street, city, region, postcode, country,
            delivery_notes: Any subset of recipient fields to update.
        """
        body: dict[str, Any] = {}
        for k, v in {
            "full_name": full_name, "phone": phone, "email": email, "company": company,
            "street": street, "city": city, "region": region, "postcode": postcode,
            "country": country, "delivery_notes": delivery_notes,
        }.items():
            if v is not None:
                body[k] = v
        return await api_patch(f"/api/v1/shipments/{shipment_id}/recipient", body)

    @mcp.tool()
    async def shipments_update_sender(
        shipment_id: str,
        full_name: str | None = None,
        phone: str | None = None,
        email: str | None = None,
        company: str | None = None,
        street: str | None = None,
        city: str | None = None,
        region: str | None = None,
        postcode: str | None = None,
        country: str | None = None,
    ) -> dict[str, Any]:
        """Partial update of shipment sender (PATCH). Same semantics as shipments_update_recipient,
        but for the sender side. Trigger a 'sender_updated' event + audit log entry.

        Args:
            shipment_id: Shipment UUID.
            Other args: partial sender fields.
        """
        body: dict[str, Any] = {}
        for k, v in {
            "full_name": full_name, "phone": phone, "email": email, "company": company,
            "street": street, "city": city, "region": region, "postcode": postcode,
            "country": country,
        }.items():
            if v is not None:
                body[k] = v
        return await api_patch(f"/api/v1/shipments/{shipment_id}/sender", body)

    @mcp.tool()
    async def shipments_add_event(
        shipment_id: str,
        event_type: str,
        location_city: str | None = None,
        location_hub: str | None = None,
        notes: str | None = None,
    ) -> dict[str, Any]:
        """Manually append a tracking event/checkpoint to a shipment's timeline. Use for
        driver-scan, carrier API hooks, or any operational note that isn't a status transition.

        Common event types:
            label_created | awaiting_pickup | picked_up | in_transit | hub_arrived | hub_departed
            | out_for_delivery | delivery_attempted | delivered | held_at_office | redirected
            | returned_to_sender | exception | customs_clearance | recipient_updated

        Args:
            shipment_id: Shipment UUID.
            event_type: One of the strings above (or any free-form descriptor).
            location_city: City where the event happened (optional).
            location_hub: Sorting hub or facility name (optional).
            notes: Free-text description.
        """
        body: dict[str, Any] = {"type": event_type}
        if location_city:
            body["location_city"] = location_city
        if location_hub:
            body["location_hub"] = location_hub
        if notes:
            body["notes"] = notes
        return await api_post(f"/api/v1/shipments/{shipment_id}/events", body)

    @mcp.tool()
    async def shipments_reschedule(
        shipment_id: str,
        new_eta: str,
        reason: str = "",
    ) -> dict[str, Any]:
        """Move the estimated delivery time of a shipment to a new value. Triggers a 'rescheduled'
        event and audit entry. Use when recipient is unavailable, weather delay, etc.

        Args:
            shipment_id: Shipment UUID.
            new_eta: New estimated delivery date-time in RFC3339 (e.g. 2026-05-14T15:00:00Z).
            reason: Human-readable reason (kept in event payload).
        """
        return await api_post(f"/api/v1/shipments/{shipment_id}/reschedule",
            {"new_eta": new_eta, "reason": reason})

    @mcp.tool()
    async def shipments_redirect(
        shipment_id: str,
        new_address: dict[str, Any],
        reason: str = "",
    ) -> dict[str, Any]:
        """Change the destination address of an in-flight shipment. Sets status='redirected',
        writes a 'redirected' event with the new city, and triggers ETA recomputation downstream.

        Refused for shipments already in delivered / returned / cancelled state.

        Args:
            shipment_id: Shipment UUID.
            new_address: Address object — at minimum {street, city}; can include full_name, phone,
                email, region, postcode, country, delivery_notes.
            reason: Human-readable reason.
        """
        return await api_post(f"/api/v1/shipments/{shipment_id}/redirect",
            {"new_address": new_address, "reason": reason})

    @mcp.tool()
    async def shipments_hold_for_pickup(
        shipment_id: str,
        office_address: str,
        reason: str = "",
    ) -> dict[str, Any]:
        """Switch a shipment to 'held_at_office' status — recipient must pick up at the carrier office.

        Args:
            shipment_id: Shipment UUID.
            office_address: Plain-text address of the pickup point (also stored as event location_hub).
            reason: Human-readable reason.
        """
        return await api_post(f"/api/v1/shipments/{shipment_id}/hold-for-pickup",
            {"office_address": office_address, "reason": reason})

    @mcp.tool()
    async def shipments_record_attempt(
        shipment_id: str,
        reason: str,
        notes: str = "",
        next_attempt_at: str | None = None,
    ) -> dict[str, Any]:
        """Record a failed delivery attempt. Auto-bumps attempt_number; on the 3rd attempt the
        shipment is automatically transitioned to 'returned_to_sender'.

        Common reasons: no_one_home | address_invalid | refused | undeliverable | locked_building.

        Args:
            shipment_id: Shipment UUID.
            reason: One of the common reasons above (or free-form short code).
            notes: Free-text detail.
            next_attempt_at: Optional RFC3339 timestamp for the next attempt.
        """
        body: dict[str, Any] = {"reason": reason, "notes": notes}
        if next_attempt_at:
            body["next_attempt_at"] = next_attempt_at
        return await api_post(f"/api/v1/shipments/{shipment_id}/record-attempt", body)

    @mcp.tool()
    async def shipments_record_delivery(
        shipment_id: str,
        signature_name: str,
        photo_url: str = "",
    ) -> dict[str, Any]:
        """Confirm delivery — transitions status to 'delivered', captures signature name and
        optional photo URL (proof of delivery). Triggers 'delivered' event.

        Args:
            shipment_id: Shipment UUID.
            signature_name: Person who signed for the package.
            photo_url: Optional URL of the proof-of-delivery photo.
        """
        return await api_post(f"/api/v1/shipments/{shipment_id}/record-delivery",
            {"signature_name": signature_name, "photo_url": photo_url})

    @mcp.tool()
    async def shipments_in_transit_summary() -> dict[str, Any]:
        """Operational dashboard: all shipments NOT yet delivered/returned/cancelled.
        Returns the shipments_list filtered to in-flight statuses with limit=200 + fetch_all=True.

        Use for 'what's in transit right now', 'pending deliveries today', 'in-flight overview',
        morning operations briefing.
        """
        all_results = []
        for status in ("created", "label_created", "awaiting_pickup", "picked_up",
                       "in_transit", "at_hub", "out_for_delivery", "delivery_attempted",
                       "held_at_office", "redirected"):
            r = await api_get_all("/api/v1/shipments", {"status": status, "limit": 200})
            payload = r.get("data") if isinstance(r, dict) else r
            if isinstance(payload, list):
                all_results.extend(payload)
        return {"data": all_results, "meta": {"total": len(all_results)}}

    @mcp.tool()
    async def shipments_reassign_carrier(
        from_carrier_id: str,
        to_carrier_id: str,
        city: str | None = None,
        statuses: list[str] | None = None,
        dry_run: bool = False,
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
            dry_run: When True, returns the list of shipment IDs that WOULD be reassigned
                without writing anything. Use dry_run=true first to preview impact.
        """
        body: dict[str, Any] = {
            "from_carrier_id": from_carrier_id,
            "to_carrier_id": to_carrier_id,
            "dry_run": dry_run,
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
