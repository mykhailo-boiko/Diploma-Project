
from typing import Any, Literal

from mcp.server.fastmcp import FastMCP

from http_client import api_get, api_post, api_put, api_patch, api_get_all
from types_mcp import (
    CarrierType, EmailAddr, ISODate, ISODateTime, Money,
    NonNegativeInt, PageLimit, PageOffset, PhoneE164, PositiveInt,
    ShipmentStatus, SortOrder, TrackingNumber, UUIDStr,
)

def register(mcp: FastMCP) -> None:

    @mcp.tool(description="List shipments with filters and pagination. For per-carrier on-time stats use analytics_carriers_performance. For shipped→cancelled forensics use analytics_quick_cancellations. For bulk reroute by city use shipments_reassign_carrier. Args: status: Filter by shipment status enum (one of 15 valid statuses). carrier_id: Carrier UUID filter. order_id: Order UUID filter. warehouse_id: Warehouse UUID filter. date_from: RFC3339 created-after datetime. date_to: RFC3339 created-before datetime. sort_by: Sort field. sort_order: 'asc' or 'desc'. limit: Page size 1..1000. offset: Page offset. fetch_all: Paginate through everything (max 5000).")
    async def shipments_list(
        status: ShipmentStatus | None = None,
        carrier_id: UUIDStr | None = None,
        order_id: UUIDStr | None = None,
        warehouse_id: UUIDStr | None = None,
        date_from: ISODateTime | None = None,
        date_to: ISODateTime | None = None,
        sort_by: Literal["created_at", "status", "order_id", "updated_at"] | None = None,
        sort_order: SortOrder | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/shipments", {
            "status": status, "carrier_id": carrier_id,
            "order_id": order_id, "warehouse_id": warehouse_id,
            "date_from": date_from, "date_to": date_to,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Get detailed information about a shipment. Args: shipment_id: Shipment UUID.")
    async def shipments_get(shipment_id: UUIDStr) -> dict[str, Any]:
        return await api_get(f"/api/v1/shipments/{shipment_id}")

    @mcp.tool(description="Create a new shipment for an order. Status starts as 'created'. Args: order_id: Order UUID this shipment fulfills. warehouse_id: Origin warehouse UUID. carrier_id: Active carrier UUID. address: Delivery address (free-form string like 'Khreshchatyk str. 22, Kyiv').")
    async def shipments_create(
        order_id: UUIDStr,
        warehouse_id: UUIDStr,
        carrier_id: UUIDStr,
        address: str,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/shipments", {
            "order_id": order_id, "warehouse_id": warehouse_id,
            "carrier_id": carrier_id, "address": address,
        })

    @mcp.tool(description="Update the status of a shipment. Valid transitions follow the 15-state postal pipeline. Prefer the semantic operations (shipments_record_delivery, shipments_record_attempt, shipments_redirect, etc.) when applicable. Args: shipment_id: Shipment UUID. status: Target status enum.")
    async def shipments_update_status(shipment_id: UUIDStr, status: ShipmentStatus) -> dict[str, Any]:
        return await api_put(f"/api/v1/shipments/{shipment_id}/status", {"status": status})

    @mcp.tool(description="Update status of multiple shipments at once. Supports partial failure. Args: updates: List of {shipment_id: UUID, status: ShipmentStatus}.")
    async def shipments_bulk_status(
        updates: list[dict[str, Any]],
    ) -> dict[str, Any]:
        return await api_post("/api/v1/shipments/bulk-status", {"updates": updates})

    @mcp.tool(description="List carriers with filters and pagination. For per-carrier on-time rate use analytics_carriers_performance. For bulk shipment reroute use shipments_reassign_carrier. Args: type: Carrier type enum. is_active: Filter by active status. name: Partial-match name. sort_by: Sort field. sort_order: 'asc' or 'desc'. limit: Page size. offset: Page offset. fetch_all: Paginate through everything.")
    async def carriers_list(
        type: CarrierType | None = None,
        is_active: bool | None = None,
        name: str | None = None,
        sort_by: Literal["created_at", "name", "type", "cost_per_km"] | None = None,
        sort_order: SortOrder | None = None,
        limit: PageLimit = 100,
        offset: PageOffset = 0,
        fetch_all: bool = False,
    ) -> dict[str, Any]:
        return await (api_get_all if fetch_all else api_get)("/api/v1/carriers", {
            "type": type, "is_active": is_active, "name": name,
            "sort_by": sort_by, "sort_order": sort_order,
            "limit": limit, "offset": offset,
        })

    @mcp.tool(description="Get a single carrier. Args: carrier_id: Carrier UUID.")
    async def carriers_get(carrier_id: UUIDStr) -> dict[str, Any]:
        return await api_get(f"/api/v1/carriers/{carrier_id}")

    @mcp.tool(description="Create a new carrier. Args: name: Carrier company name. type: Transport type enum. cost_per_km: Cost per kilometer in currency units (>= 0).")
    async def carriers_create(
        name: str,
        type: CarrierType,
        cost_per_km: Money,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/carriers", {
            "name": name, "type": type, "cost_per_km": cost_per_km,
        })

    @mcp.tool(description="Update an existing carrier. Args: carrier_id: Carrier UUID. name: Updated carrier name. type: Transport type enum. cost_per_km: Updated cost per km. is_active: Active flag.")
    async def carriers_update(
        carrier_id: UUIDStr,
        name: str,
        type: CarrierType,
        cost_per_km: Money,
        is_active: bool = True,
    ) -> dict[str, Any]:
        return await api_put(f"/api/v1/carriers/{carrier_id}", {
            "name": name, "type": type,
            "cost_per_km": cost_per_km, "is_active": is_active,
        })

    @mcp.tool(description="Full postal tracking record by tracking number (CO-YYYY-XXXXXX). Returns: {shipment, events[], delivery_attempts[]}. Args: tracking_number: Code from shipment.tracking_number (NOT a UUID).")
    async def shipments_tracking(tracking_number: TrackingNumber) -> dict[str, Any]:
        return await api_get(f"/api/v1/tracking/{tracking_number}")

    @mcp.tool(description="Full timeline by shipment UUID — events + attempts + shipment row. Prefer shipments_tracking when you have the tracking_number. Args: shipment_id: Shipment UUID.")
    async def shipments_timeline(shipment_id: UUIDStr) -> dict[str, Any]:
        return await api_get(f"/api/v1/shipments/{shipment_id}/timeline")

    @mcp.tool(description="PATCH recipient fields on a shipment (only provided fields change). Allowed until status=delivered. Phone must be E.164 (+380501112233). Email must be valid. Args: shipment_id: Shipment UUID. full_name, phone, email, company, street, city, region, postcode, country, delivery_notes: Any subset of recipient fields.")
    async def shipments_update_recipient(
        shipment_id: UUIDStr,
        full_name: str | None = None,
        phone: PhoneE164 | None = None,
        email: EmailAddr | None = None,
        company: str | None = None,
        street: str | None = None,
        city: str | None = None,
        region: str | None = None,
        postcode: str | None = None,
        country: str | None = None,
        delivery_notes: str | None = None,
    ) -> dict[str, Any]:
        body: dict[str, Any] = {}
        for k, v in {
            "full_name": full_name, "phone": phone, "email": email, "company": company,
            "street": street, "city": city, "region": region, "postcode": postcode,
            "country": country, "delivery_notes": delivery_notes,
        }.items():
            if v is not None:
                body[k] = v
        return await api_patch(f"/api/v1/shipments/{shipment_id}/recipient", body)

    @mcp.tool(description="PATCH sender fields. Same semantics as shipments_update_recipient. Args: shipment_id: Shipment UUID. Other args: any subset of sender fields.")
    async def shipments_update_sender(
        shipment_id: UUIDStr,
        full_name: str | None = None,
        phone: PhoneE164 | None = None,
        email: EmailAddr | None = None,
        company: str | None = None,
        street: str | None = None,
        city: str | None = None,
        region: str | None = None,
        postcode: str | None = None,
        country: str | None = None,
    ) -> dict[str, Any]:
        body: dict[str, Any] = {}
        for k, v in {
            "full_name": full_name, "phone": phone, "email": email, "company": company,
            "street": street, "city": city, "region": region, "postcode": postcode,
            "country": country,
        }.items():
            if v is not None:
                body[k] = v
        return await api_patch(f"/api/v1/shipments/{shipment_id}/sender", body)

    @mcp.tool(description="Append a tracking event to a shipment's timeline. Args: shipment_id: Shipment UUID. event_type: One of valid event type enum values. location_city: City where event happened (optional). location_hub: Sorting hub or facility (optional). notes: Free-text description.")
    async def shipments_add_event(
        shipment_id: UUIDStr,
        event_type: Literal[
            "label_created", "awaiting_pickup", "picked_up", "in_transit",
            "hub_arrived", "hub_departed", "out_for_delivery", "delivery_attempted",
            "delivered", "held_at_office", "redirected", "returned_to_sender",
            "exception", "customs_clearance", "recipient_updated",
        ],
        location_city: str | None = None,
        location_hub: str | None = None,
        notes: str | None = None,
    ) -> dict[str, Any]:
        body: dict[str, Any] = {"type": event_type}
        if location_city:
            body["location_city"] = location_city
        if location_hub:
            body["location_hub"] = location_hub
        if notes:
            body["notes"] = notes
        return await api_post(f"/api/v1/shipments/{shipment_id}/events", body)

    @mcp.tool(description="Move the estimated delivery time to a new value. Args: shipment_id: Shipment UUID. new_eta: New estimated delivery datetime in RFC3339 (e.g. '2026-05-14T15:00:00Z'). reason: Human-readable reason.")
    async def shipments_reschedule(
        shipment_id: UUIDStr,
        new_eta: ISODateTime,
        reason: str = "",
    ) -> dict[str, Any]:
        return await api_post(f"/api/v1/shipments/{shipment_id}/reschedule",
            {"new_eta": new_eta, "reason": reason})

    @mcp.tool(description="Change destination address of an in-flight shipment. Refused for delivered / returned / cancelled shipments. Args: shipment_id: Shipment UUID. new_address: Address dict — at least {street, city}; optional fields full_name, phone (E.164), email, region, postcode, country, delivery_notes. reason: Human-readable reason.")
    async def shipments_redirect(
        shipment_id: UUIDStr,
        new_address: dict[str, Any],
        reason: str = "",
    ) -> dict[str, Any]:
        return await api_post(f"/api/v1/shipments/{shipment_id}/redirect",
            {"new_address": new_address, "reason": reason})

    @mcp.tool(description="Switch shipment to 'held_at_office' status. Args: shipment_id: Shipment UUID. office_address: Pickup point address. reason: Human-readable reason.")
    async def shipments_hold_for_pickup(
        shipment_id: UUIDStr,
        office_address: str,
        reason: str = "",
    ) -> dict[str, Any]:
        return await api_post(f"/api/v1/shipments/{shipment_id}/hold-for-pickup",
            {"office_address": office_address, "reason": reason})

    @mcp.tool(description="Record a failed delivery attempt. Auto-bumps attempt_number; 3rd → returned_to_sender. Args: shipment_id: Shipment UUID. reason: Reason enum. notes: Free-text detail. next_attempt_at: Optional RFC3339 datetime.")
    async def shipments_record_attempt(
        shipment_id: UUIDStr,
        reason: Literal["no_one_home", "address_invalid", "refused", "undeliverable", "locked_building", "other"],
        notes: str = "",
        next_attempt_at: ISODateTime | None = None,
    ) -> dict[str, Any]:
        body: dict[str, Any] = {"reason": reason, "notes": notes}
        if next_attempt_at:
            body["next_attempt_at"] = next_attempt_at
        return await api_post(f"/api/v1/shipments/{shipment_id}/record-attempt", body)

    @mcp.tool(description="Confirm delivery → transitions status to 'delivered'. Args: shipment_id: Shipment UUID. signature_name: Person who signed. photo_url: Optional proof-of-delivery URL.")
    async def shipments_record_delivery(
        shipment_id: UUIDStr,
        signature_name: str,
        photo_url: str = "",
    ) -> dict[str, Any]:
        return await api_post(f"/api/v1/shipments/{shipment_id}/record-delivery",
            {"signature_name": signature_name, "photo_url": photo_url})

    @mcp.tool(description="All shipments NOT yet delivered/returned/cancelled — in-flight overview.")
    async def shipments_in_transit_summary() -> dict[str, Any]:
        all_results = []
        for status in ("created", "label_created", "awaiting_pickup", "picked_up",
                       "in_transit", "at_hub", "out_for_delivery", "delivery_attempted",
                       "held_at_office", "redirected"):
            r = await api_get_all("/api/v1/shipments", {"status": status, "limit": 200})
            payload = r.get("data") if isinstance(r, dict) else r
            if isinstance(payload, list):
                all_results.extend(payload)
        return {"data": all_results, "meta": {"total": len(all_results)}}

    @mcp.tool(description="Bulk-reassign shipments from one carrier to another. Only in-motion shipments are affected (default created/picked_up/in_transit). Args: from_carrier_id: Source carrier UUID. to_carrier_id: Target carrier UUID (must differ). city: Optional destination city filter. statuses: Optional list of statuses to touch. dry_run: If True, preview impact without writing.")
    async def shipments_reassign_carrier(
        from_carrier_id: UUIDStr,
        to_carrier_id: UUIDStr,
        city: str | None = None,
        statuses: list[ShipmentStatus] | None = None,
        dry_run: bool = False,
    ) -> dict[str, Any]:
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

    @mcp.tool(description="Calculate route distance/duration/cost. Args: origin: Origin city/location name. destination: Destination city/location name. carrier_id: Carrier UUID for cost calc.")
    async def routes_calculate(
        origin: str,
        destination: str,
        carrier_id: UUIDStr,
    ) -> dict[str, Any]:
        return await api_post("/api/v1/routes/calculate", {
            "origin": origin, "destination": destination, "carrier_id": carrier_id,
        })

    @mcp.tool(description="Delivery performance metrics: total delivered, on-time, late, on-time rate.")
    async def logistics_performance() -> dict[str, Any]:
        return await api_get("/api/v1/logistics/performance")
