# MCP Tools Reference

The MCP Orchestrator exposes **63 tools** across 6 service domains. Tools are registered via [FastMCP](https://github.com/modelcontextprotocol/python-sdk) and called by the Gemini LLM through the MCP Host.

## RBAC Tool Access

Tools are filtered by user role **before** being sent to the LLM:

| Role | Allowed Tool Prefixes |
|------|----------------------|
| **Admin** | All tools (unrestricted) |
| **Operator** | `orders_*`, `notifications_*` |
| **Warehouse Manager** | `products_*`, `warehouses_*`, `stock_*`, `inventory_*`, `orders_*` |
| **Logistics Manager** | `shipments_*`, `carriers_*`, `routes_*`, `logistics_*`, `orders_*` |
| **Analyst** | `analytics_*` |

**Common tools** (all roles): `users_login`, `users_register`, `users_refresh_token`, `users_password_reset`, `users_password_reset_confirm`, `users_me`, `users_update_profile`

---

## Orders (7 tools)

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

## Shipments (5 tools)

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

## Analytics (10 tools)

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

---

## Notifications (7 tools)

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

## Tool Count Summary

| Domain | Tools |
|--------|-------|
| Orders | 7 |
| Products | 5 |
| Warehouses | 4 |
| Stock | 7 |
| Inventory Report | 1 |
| Shipments | 5 |
| Carriers | 4 |
| Routes | 1 |
| Logistics Performance | 1 |
| Analytics | 10 |
| Notifications | 7 |
| Users | 11 |
| **Total** | **63** |
