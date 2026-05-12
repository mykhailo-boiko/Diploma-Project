# MCP Tools Reference

The MCP Orchestrator exposes **93 tools** across 7 service domains. Tools are registered via [FastMCP](https://github.com/modelcontextprotocol/python-sdk) and called by the Gemini LLM through the MCP Host.

## RBAC Tool Access

Tools are filtered by user role **before** being sent to the LLM:

| Role | Allowed Tool Prefixes |
|------|----------------------|
| **Admin** | All tools (unrestricted) |
| **Operator** | `orders_*`, `notifications_*` |
| **Warehouse Manager** | `products_*`, `warehouses_*`, `stock_*`, `inventory_*`, `orders_*` |
| **Logistics Manager** | `shipments_*`, `carriers_*`, `routes_*`, `logistics_*`, `orders_*` |
| **Analyst** | `analytics_*` |

**Common tools** (all roles): `users_me`, `users_update_profile`

**Admin-only domains**: `simulator_*` (live traffic generator), `audit_query`, `audit_trace_by_entity`, `users_register`, `users_password_reset` — visible only when `role == "admin"` because the prefix is not in any non-admin role's allow-list.

## Strict Parameter Typing

All tools use strict Pydantic types via `typing.Annotated` so FastMCP emits precise JSON Schemas
that Gemini honours. The `string` / `int` columns below describe the JSON wire format; at runtime
Pydantic validators enforce specific formats and emit LLM-friendly error messages on mismatch.

| Annotated type | Wire format | Constraint |
|----------------|-------------|------------|
| `UUIDStr` | string | 8-4-4-4-12 hex UUID |
| `TrackingNumber` | string | `CO-YYYY-XXXXXX` pattern |
| `ISODate` | string | `YYYY-MM-DD` ISO 8601 date |
| `ISODateTime` | string | RFC3339 datetime |
| `PhoneE164` | string | `+CCC...` (7-15 digits) |
| `EmailAddr` | string | `local@domain.tld` |
| `PositiveInt` | int | > 0, <= 1_000_000 |
| `NonNegativeInt` | int | >= 0 |
| `PageLimit` | int | 1..1000 |
| `Money` | number | 0..10M |
| `OrderStatus`, `ShipmentStatus`, … | string | enum-restricted via `Literal[...]` |

See [`docs/typed-tools.md`](typed-tools.md) for the complete catalog and behaviour, and
[`docs/error-codes.md`](error-codes.md) for the structured error envelope returned on validation failure.

---

## Orders (10 tools)

### orders_list
List orders with optional filters and pagination.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `status` | string | No | Filter: pending, confirmed, processing, shipped, delivered, completed, cancelled, returned |
| `date_from` | string | No | RFC3339 date (e.g., 2026-01-01T00:00:00Z) |
| `date_to` | string | No | RFC3339 date |
| `customer_name` | string | No | Partial match |
| `sort_by` | string | No | created_at, total_amount, status, customer_name |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### orders_get
Get detailed information about a specific order including its line items.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `order_id` | string | Yes | Order UUID |

### orders_create
Create a new order with line items. Starts in 'pending' status.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `customer_name` | string | Yes | Customer name |
| `items` | list[dict] | Yes | Each: `{product_id, name, quantity, unit_price}` |

### orders_update_status
Update order status. Valid: pending->confirmed->processing->shipped->delivered->completed.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `order_id` | string | Yes | Order UUID |
| `status` | string | Yes | New status |

### orders_cancel
Cancel an order with a reason.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `order_id` | string | Yes | Order UUID |
| `reason` | string | Yes | Cancellation reason |

### orders_search
Search orders by customer name or order ID. Minimum 2 characters.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `query` | string | Yes | Search string (min 2 chars) |

### orders_stats
Get order statistics: total count, total revenue, and breakdown by status.

*No parameters.*

### orders_bulk_update_status
Update status for many orders in ONE server-side call with per-order success/failure report.

| Param | Type | Description |
|-------|------|-------------|
| order_ids | list[str] | UUIDs (max 500 per call) |
| status | str | Target status |
| note | str (optional) | Service comment (stored in cancel_reason) |
| dry_run | bool (default false) | Preview without writes |

Returns `{total, dry_run, updated_ids, successes[], failures[]}`.

### orders_sales_by_product
Per-SKU sales aggregate for a date range: units_sold, revenue, order_count, daily_demand.

| Param | Type | Description |
|-------|------|-------------|
| date_from | str | YYYY-MM-DD inclusive |
| date_to | str | YYYY-MM-DD inclusive |
| statuses | str (optional) | CSV order statuses to include |
| limit | int (default 0) | Cap on rows |

### orders_customer_summary
Per-customer aggregate with lifetime + window metrics + new_in_window flag.

| Param | Type | Description |
|-------|------|-------------|
| date_from, date_to | str (optional) | Optional window for orders_in_window/revenue_in_window |
| only_new | bool (default false) | Customers whose first-ever order is in the window |
| sort_by | str | revenue / revenue_in_window / orders / last_order / first_order |
| sort_order | str | asc \| desc |
| limit | int | Cap on rows |

---

## Products (5 tools)

### products_list
List products with optional filters and pagination.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sku` | string | No | Exact SKU match |
| `name` | string | No | Partial name match |
| `category` | string | No | Exact category match |
| `sort_by` | string | No | created_at, name, sku, category, unit_price |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### products_get
Get product details.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product UUID |

### products_create
Create a new product. SKU must be unique.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `sku` | string | Yes | Unique SKU |
| `name` | string | Yes | Product name |
| `description` | string | No | Description |
| `category` | string | No | Category |
| `unit_price` | float | No | Price per unit |

### products_update
Update an existing product.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product UUID |
| `name` | string | Yes | Product name |
| `description` | string | No | Description |
| `category` | string | No | Category |
| `unit_price` | float | No | Price per unit |

### products_delete
Soft-delete a product.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product UUID |

---

## Warehouses (4 tools)

### warehouses_list
List warehouses with optional filters.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | No | Partial name match |
| `sort_by` | string | No | created_at, name |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### warehouses_get
Get warehouse details.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `warehouse_id` | string | Yes | Warehouse UUID |

### warehouses_create
Create a new warehouse.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Warehouse name |
| `address` | string | No | Address |

### warehouses_update
Update an existing warehouse.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `warehouse_id` | string | Yes | Warehouse UUID |
| `name` | string | Yes | Warehouse name |
| `address` | string | No | Address |
| `is_active` | bool | No | Default: true |

---

## Stock (7 tools)

### stock_list
List stock levels. Shows quantity, reserved, and available per product-warehouse.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | No | Filter by product |
| `warehouse_id` | string | No | Filter by warehouse |
| `sort_by` | string | No | updated_at, product_id, warehouse_id, quantity, available |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### stock_reserve
Reserve stock for an order. Uses atomic locking (SELECT FOR UPDATE) to prevent overselling.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product UUID |
| `warehouse_id` | string | Yes | Warehouse UUID |
| `quantity` | int | Yes | Units to reserve |
| `reference` | string | No | Reference (e.g., order ID) |

### stock_release
Release previously reserved stock.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product UUID |
| `warehouse_id` | string | Yes | Warehouse UUID |
| `quantity` | int | Yes | Units to release |
| `reference` | string | No | Reference ID |

### stock_adjust
Adjust stock quantity (inbound, outbound, or manual adjustment).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product UUID |
| `warehouse_id` | string | Yes | Warehouse UUID |
| `quantity` | int | Yes | Units to adjust (positive) |
| `type` | string | Yes | inbound, outbound, adjustment |
| `reference` | string | No | Reference ID |

### stock_movements
Get stock movement history.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `stock_id` | string | No | Filter by stock record |
| `product_id` | string | No | Filter by product |
| `warehouse_id` | string | No | Filter by warehouse |
| `type` | string | No | reserve, release, inbound, outbound, adjustment |
| `date_from` | string | No | RFC3339 date |
| `date_to` | string | No | RFC3339 date |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### stock_low
Get products with stock below their minimum threshold.

*No parameters.*

### stock_threshold_update
Update minimum stock threshold for a product-warehouse pair.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `product_id` | string | Yes | Product UUID |
| `warehouse_id` | string | Yes | Warehouse UUID |
| `threshold` | int | Yes | Minimum threshold (0 = disabled) |

---

## Inventory Report (1 tool)

### inventory_report
Get comprehensive inventory report with totals by warehouse and category.

*No parameters.*

---

## Shipments (17 tools)

### shipments_list
List shipments with optional filters.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `status` | string | No | created, picked_up, in_transit, delivered, failed, returned |
| `carrier_id` | string | No | Filter by carrier |
| `order_id` | string | No | Filter by order |
| `warehouse_id` | string | No | Filter by warehouse |
| `date_from` | string | No | RFC3339 date |
| `date_to` | string | No | RFC3339 date |
| `sort_by` | string | No | created_at, status, order_id |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### shipments_get
Get shipment details.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `shipment_id` | string | Yes | Shipment UUID |

### shipments_create
Create a new shipment. Status starts as 'created'.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `order_id` | string | Yes | Order UUID |
| `warehouse_id` | string | Yes | Source warehouse UUID |
| `carrier_id` | string | Yes | Carrier UUID |
| `address` | string | Yes | Delivery address |

### shipments_update_status
Update shipment status. Valid: created->picked_up->in_transit->delivered/failed->returned.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `shipment_id` | string | Yes | Shipment UUID |
| `status` | string | Yes | New status |

### shipments_bulk_status
Bulk status update. Supports partial failure (207 Multi-Status).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `updates` | list[dict] | Yes | Each: `{shipment_id, status}` |

### shipments_tracking
Full postal tracking record by human-readable tracking number (e.g. `CO-2026-K7H2P9`). Returns `{shipment, events[], delivery_attempts[]}`.

| Param | Type | Description |
|-------|------|-------------|
| tracking_number | str | Tracking code (not the UUID) |

### shipments_timeline
Same as `shipments_tracking` but by internal UUID.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Internal UUID |

### shipments_update_recipient
PATCH semantics — only provided fields change. Phone E.164 validated. Allowed until status=delivered.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Shipment UUID |
| full_name, phone, email, company, street, city, region, postcode, country, delivery_notes | str (optional) | Any subset |

### shipments_update_sender
PATCH semantics for sender side. Same shape as `shipments_update_recipient`.

### shipments_add_event
Append a manual tracking checkpoint to the timeline.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Shipment UUID |
| event_type | str | label_created / picked_up / hub_arrived / out_for_delivery / customs_clearance / exception / etc. |
| location_city, location_hub, notes | str (optional) | Context |

### shipments_reschedule
Move estimated_delivery_at; writes a `rescheduled` event.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Shipment UUID |
| new_eta | str | RFC3339 timestamp |
| reason | str (optional) | Human-readable reason |

### shipments_redirect
Change destination address mid-flight. Sets status=`redirected`, writes a `redirected` event. Refused if delivered/returned/cancelled.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Shipment UUID |
| new_address | dict | At minimum `{street, city}`; may include phone/email/postcode/etc. |
| reason | str (optional) | Reason |

### shipments_hold_for_pickup
Switch to `held_at_office` — recipient must pick up.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Shipment UUID |
| office_address | str | Pickup-point address |
| reason | str (optional) | Reason |

### shipments_record_attempt
Log a failed delivery attempt. Auto-bumps attempt_number; on the 3rd attempt status → `returned_to_sender` automatically.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Shipment UUID |
| reason | str | no_one_home / address_invalid / refused / undeliverable / locked_building |
| notes | str (optional) | Free-text |
| next_attempt_at | str (optional) | RFC3339 |

### shipments_record_delivery
Confirm delivery: status → `delivered`, captures signature + photo URL.

| Param | Type | Description |
|-------|------|-------------|
| shipment_id | str | Shipment UUID |
| signature_name | str | Person who signed |
| photo_url | str (optional) | Proof-of-delivery URL |

### shipments_in_transit_summary
Operational dashboard — all shipments NOT yet delivered/returned/cancelled (label_created → out_for_delivery + redirected/held).

*No parameters.* Returns aggregated list across in-flight statuses.

### shipments_reassign_carrier
Bulk-reassign shipments from one carrier to another, optionally filtered by destination city.

| Param | Type | Description |
|-------|------|-------------|
| from_carrier_id | str | Source carrier UUID |
| to_carrier_id | str | Target carrier UUID |
| city | str (optional) | Filter by destination city (ILIKE) |
| statuses | list[str] (optional) | Defaults to `[created, picked_up, in_transit]` |
| dry_run | bool (default false) | Preview only |

---

## Carriers (4 tools)

### carriers_list
List carriers with optional filters.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | string | No | ground, air, sea |
| `is_active` | string | No | true, false |
| `name` | string | No | Partial name match |
| `sort_by` | string | No | created_at, name, type, cost_per_km |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### carriers_get
Get carrier details.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `carrier_id` | string | Yes | Carrier UUID |

### carriers_create
Create a new carrier.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | Yes | Carrier name |
| `type` | string | Yes | ground, air, sea |
| `cost_per_km` | float | Yes | Cost per kilometer |

### carriers_update
Update an existing carrier.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `carrier_id` | string | Yes | Carrier UUID |
| `name` | string | Yes | Carrier name |
| `type` | string | Yes | ground, air, sea |
| `cost_per_km` | float | Yes | Cost per kilometer |
| `is_active` | bool | No | Default: true |

---

## Routes (1 tool)

### routes_calculate
Calculate a delivery route with distance, duration, and cost (mock implementation).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `origin` | string | Yes | Origin location name |
| `destination` | string | Yes | Destination location name |
| `carrier_id` | string | Yes | Carrier UUID (for cost calculation) |

Returns: `{id, origin, destination, distance_km, duration_h, cost}`

---

## Logistics Performance (1 tool)

### logistics_performance
Get delivery performance metrics.

*No parameters.*

Returns: `{total_delivered, on_time, late, on_time_rate}`

---

## Analytics (16 tools)

### analytics_sales
Get daily sales data for a date range.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

### analytics_sales_summary
Aggregated sales: revenue, order_count, avg_order_value.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

### analytics_sales_trends
Sales trends aggregated by day or week.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |
| `granularity` | string | No | day (default), week |

### analytics_inventory
Daily inventory snapshots.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

### analytics_inventory_summary
Aggregated inventory: stock levels, reserved, available, low-stock count, turnover rate.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

### analytics_logistics
Daily logistics metrics.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

### analytics_logistics_performance
Logistics performance: shipment counts, delivery rate, on-time rate, avg delivery hours.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

### analytics_anomalies
Rule-based anomaly detection across sales, inventory, and logistics.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

Anomaly rules:
- Sales: revenue > 2 sigma (warning) or 3 sigma (critical), zero-order days
- Logistics: failure rate > 20% (critical), on-time rate < 80% (warning)
- Inventory: low-stock > 10% of products (warning)

### analytics_optimization
Reorder recommendations based on demand analysis.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

Formula: `reorder_point = avg_daily_demand * lead_time(7d) + safety_stock(1.5x)`

### analytics_report
Generate a custom analytics report.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `report_type` | string | Yes | sales, inventory, logistics, full |
| `date_from` | string | Yes | YYYY-MM-DD |
| `date_to` | string | Yes | YYYY-MM-DD |

### analytics_carriers_performance
Per-carrier on-time rate + worst-cities breakdown. Sorted ASC by on_time_rate (worst first).

| Param | Type | Description |
|-------|------|-------------|
| date_from, date_to | str | YYYY-MM-DD inclusive |
| sla_hours | int (default 168) | Cutoff for on-time vs late |
| worst_cities | int (default 5) | Top N cities per carrier |

### analytics_quick_cancellations
Forensic: orders cancelled within `max_minutes` after their shipment was created, grouped by carrier × destination city.

| Param | Type | Description |
|-------|------|-------------|
| date_from, date_to | str | Window YYYY-MM-DD |
| max_minutes | int (default 60) | Cutoff for "quick" cancellation |

### analytics_rebalancing_recommendations
Cross-warehouse stock rebalancing with explicit cost model. Best donor per (SKU, acceptor), ranked by net_benefit.

| Param | Type | Description |
|-------|------|-------------|
| overstock_multiplier | float (default 3.0) | Donor if quantity > min_threshold × this |
| holding_daily_rate | float (default 0.0005) | Carrying cost rate per day |
| holding_horizon_days | int (default 30) | Holding-savings amortization |
| transfer_base_fee | float (default 50.0) | Fixed dispatch fee |
| transfer_per_unit | float (default 1.5) | Variable cost per unit |
| include_unprofitable | bool (default false) | Include negative-ROI rows |
| limit | int (default 50) | Cap on rows |

### analytics_period_comparison
Compare a metric between two arbitrary date windows. Returns absolute delta, % change, direction, significance label.

| Param | Type | Description |
|-------|------|-------------|
| metric | str | revenue / order_count / aov / cancellation_rate / on_time_rate / shipment_count / low_stock_count |
| a_from, a_to | str | Baseline window (YYYY-MM-DD) |
| b_from, b_to | str | Comparison window |
| a_label, b_label | str (optional) | Human-readable labels (e.g. "Q1 2026") |

### analytics_forecast
Server-side time-series projection with confidence band + backtest MAPE.

| Param | Type | Description |
|-------|------|-------------|
| metric | str | revenue / order_count / shipment_count |
| horizon_days | int (default 14) | Days to project forward |
| history_days | int (default 30) | Trailing window used to fit |
| method | str (default linear) | linear / rolling-avg / ets-simple |

### analytics_what_if
Counterfactual simulator with assumptions and qualitative confidence.

| Param | Type | Description |
|-------|------|-------------|
| kind | str | carrier_drop / capacity_increase / price_change / promo_burst |
| params | dict | Scenario-specific (see tool docstring) |

Returns `{scenario, baseline, projected, delta, delta_percent, assumptions[], confidence_qualitative, human_summary}`.

---

## Customers (1 tool)

### customers_profile_360
Executive-grade single-customer dossier in ONE call: lifetime aggregates, churn risk, top categories, recent orders, status mix.

| Param | Type | Description |
|-------|------|-------------|
| customer_name | str | Exact customer name |
| recent_n | int (default 5) | Most-recent orders to include |
| top_categories_n | int (default 5) | Top spending categories |

Returns `{customer_name, first_order_date, last_order_date, lifetime_value, order_count, avg_order_value, days_since_last_order, median_inter_order_days, churn_risk_score, status_breakdown, top_categories[], recent_orders[], is_new_customer_90_days}`.

---

## Audit (2 tools)

### audit_query
Query the audit trail of write operations across the platform.

| Param | Type | Description |
|-------|------|-------------|
| actor_email | str (optional) | Filter to one actor |
| action | str (optional) | Exact action name (e.g. `orders.bulk_update_status`) |
| entity_id | UUIDStr (optional) | Entity UUID touched by the action |
| date_from, date_to | ISODate (optional) | YYYY-MM-DD window |
| limit | PositiveInt (default 50, max 500) | Cap on rows |

Each entry: `{actor_user_id, actor_email, actor_role, service_name, action, entity_type, entity_ids, params_snip, result_status, success_count, failure_count, error_message, trace_id, created_at}`.

### audit_trace_by_entity
Return the full chronological audit trail for a single entity (order, shipment, product, etc.).

| Param | Type | Description |
|-------|------|-------------|
| entity_id | UUIDStr | UUID of the entity to trace |
| limit | PositiveInt (default 200) | Max events |

Returns `{entity_id, total, trace_ids[], events[]}`. The `trace_ids[]` field links each audit row to the corresponding Logfire span via `X-Trace-ID` propagation across services.

---

## Notifications (8 tools)

### notifications_list
List notifications for the current user.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | string | No | order_created, order_updated, order_cancelled, low_stock, stock_changed, shipment_created, shipment_updated, system |
| `sort_by` | string | No | created_at, type, status |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### notifications_create
Create a notification (admin only).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user_id` | string | Yes | Recipient user UUID |
| `type` | string | Yes | Notification type |
| `title` | string | Yes | Title |
| `message` | string | Yes | Message body |

### notifications_mark_read
Mark a notification as read.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `notification_id` | string | Yes | Notification UUID |

### notifications_unread_count
Get unread notification count for current user.

*No parameters.*

### notifications_unread_counts
Admin-only: unread counts for every user, enriched with email/name/role. Filterable by role.

| Param | Type | Description |
|-------|------|-------------|
| role | str (optional) | admin / warehouse_manager / logistics_manager / analyst / operator |

Returns rows sorted by unread_count descending.

### notifications_preferences_get
Get notification preferences (per-type channel toggles).

*No parameters.*

### notifications_preferences_update
Update notification preferences for a type.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `type` | string | Yes | Notification type |
| `in_app` | bool | No | Default: true |
| `email` | bool | No | Default: true |
| `sms` | bool | No | Default: false |

### notifications_bulk
Send notification to multiple users (admin only).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user_ids` | list[str] | Yes | Recipient UUIDs |
| `type` | string | Yes | Notification type |
| `title` | string | Yes | Title |
| `message` | string | Yes | Message body |

---

## Users (11 tools)

### users_login
Authenticate and get JWT tokens.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `email` | string | Yes | Email address |
| `password` | string | Yes | Password |

### users_register
Register a new user (admin only).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `email` | string | Yes | Unique email |
| `password` | string | Yes | Password |
| `first_name` | string | Yes | First name |
| `last_name` | string | Yes | Last name |
| `role` | string | Yes | admin, warehouse_manager, logistics_manager, analyst, operator |

### users_refresh_token
Refresh access token.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `refresh_token` | string | Yes | Refresh token from login |

### users_password_reset
Request password reset (mock email adapter).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `email` | string | Yes | Account email |

### users_password_reset_confirm
Confirm password reset with token.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `token` | string | Yes | Reset token (from email) |
| `new_password` | string | Yes | New password |

### users_me
Get current user profile.

*No parameters.*

### users_update_profile
Update current user profile (cannot change role).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `first_name` | string | Yes | First name |
| `last_name` | string | Yes | Last name |
| `email` | string | Yes | Email |

### users_list
List all users (admin only).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `role` | string | No | Filter by role |
| `email` | string | No | Partial email match |
| `name` | string | No | Partial name match |
| `sort_by` | string | No | created_at, email, first_name, last_name, role |
| `sort_order` | string | No | asc, desc |
| `limit` | int | No | Default: 20 |
| `offset` | int | No | Default: 0 |

### users_create
Create a new user with role (admin only).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `email` | string | Yes | Unique email |
| `password` | string | Yes | Password |
| `first_name` | string | Yes | First name |
| `last_name` | string | Yes | Last name |
| `role` | string | Yes | User role |

### users_update
Update user including role (admin only).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user_id` | string | Yes | User UUID |
| `first_name` | string | Yes | First name |
| `last_name` | string | Yes | Last name |
| `email` | string | Yes | Email |
| `role` | string | Yes | User role |

### users_delete
Soft-delete a user (admin only).

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `user_id` | string | Yes | User UUID |

---

## Simulator (5 tools, admin-only)

Controls for the live traffic-generation service. The simulator continuously creates orders, advances shipments through the 15-state postal pipeline, fluctuates inventory and emits notifications so the platform behaves like a 24/7 production system. All tools require `admin` role.

### `simulator_status`

Returns the current state of the live simulator: `enabled`, `scenario`, `speed`, `uptime_secs`, and counters (`orders_created`, `orders_progressed`, `orders_cancelled`, `shipments_advanced`, `shipment_events`, `stock_adjustments`, `notifications_sent`, `errors`).

No parameters.

### `simulator_start`

Start the simulator.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `scenario` | string | No (default `steady`) | One of `idle`, `steady`, `holiday_spike`, `carrier_failure`, `demand_surge` |
| `speed` | number | No (default `1.0`) | Time-acceleration multiplier (0 < speed ≤ 100, typical: 1, 5, 10, 25, 50) |

### `simulator_stop`

Pause all actors. Counters retained for inspection.

No parameters.

### `simulator_set_speed`

Adjust the time multiplier without restart.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `speed` | number | Yes | New multiplier (0 < speed ≤ 100) |

### `simulator_set_scenario`

Switch active scenario without restart.

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `scenario` | string | Yes | One of `idle`, `steady`, `holiday_spike`, `carrier_failure`, `demand_surge` |

---

## Tool Count Summary

| Domain | Tools |
|--------|-------|
| Orders | 10 |
| Products | 5 |
| Warehouses | 4 |
| Stock | 7 |
| Inventory Report | 1 |
| Shipments | 17 |
| Carriers | 4 |
| Routes | 1 |
| Logistics Performance | 1 |
| Analytics | 16 |
| Customers | 1 |
| Audit | 2 |
| Notifications | 8 |
| Users | 11 |
| Simulator | 5 |
| **Total** | **93** |
