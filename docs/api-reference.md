# API Reference

All endpoints are accessed through the **API Gateway** at `http://localhost:8080`.

Authentication: Include `Authorization: Bearer <jwt_token>` header. Endpoints under `/api/v1/auth/*` are public.

Standard response format:
```json
{
  "data": { ... },
  "meta": { "total": 100, "limit": 20, "offset": 0 },
  "error": null
}
```

---

## Authentication (user-service :8001)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/auth/login` | Public | Login with email/password, returns access + refresh JWT |
| `POST` | `/api/v1/auth/register` | Admin | Register a new user with role |
| `POST` | `/api/v1/auth/refresh` | Public | Refresh access token using refresh token |
| `POST` | `/api/v1/auth/password-reset` | Public | Request password reset (mock email) |
| `POST` | `/api/v1/auth/password-reset/confirm` | Public | Confirm reset with token + new password |

### Login Request/Response

```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email": "admin@chainorchestra.local", "password": "admin123"}'
```

```json
{
  "data": {
    "access_token": "eyJhbGciOi...",
    "refresh_token": "eyJhbGciOi...",
    "user": {
      "id": "uuid",
      "email": "admin@chainorchestra.local",
      "first_name": "Admin",
      "last_name": "User",
      "role": "admin"
    }
  }
}
```

---

## Users (user-service :8001)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/v1/users/me` | Any | Get current user profile |
| `PUT` | `/api/v1/users/me` | Any | Update profile (cannot change role) |
| `GET` | `/api/v1/users` | Admin | List users with filters and pagination |
| `POST` | `/api/v1/users` | Admin | Create a new user |
| `PUT` | `/api/v1/users/:id` | Admin | Update user including role |
| `DELETE` | `/api/v1/users/:id` | Admin | Soft delete user |

**Query parameters** (GET /api/v1/users): `role`, `email`, `name`, `sort_by`, `sort_order`, `limit`, `offset`

---

## Orders (order-service :8002)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/orders` | Auth | Create order with line items |
| `GET` | `/api/v1/orders` | Auth | List orders with filters |
| `GET` | `/api/v1/orders/:id` | Auth | Get order details with items |
| `PUT` | `/api/v1/orders/:id/status` | Auth | Update order status |
| `POST` | `/api/v1/orders/:id/cancel` | Auth | Cancel order with reason |
| `GET` | `/api/v1/orders/search` | Auth | Search by customer name / order ID |
| `GET` | `/api/v1/orders/stats` | Auth | Order statistics by status |

**Query parameters** (GET /api/v1/orders): `status`, `date_from`, `date_to`, `customer_name`, `sort_by`, `sort_order`, `limit`, `offset`

**Order Status Workflow:**
```
pending → confirmed → processing → shipped → delivered → completed
                                                         ↗
Any status ──────────────────────────────────→ cancelled
shipped ──────────────────────────────────────→ returned
```

### Create Order

```bash
curl -X POST http://localhost:8080/api/v1/orders \
  -H 'Authorization: Bearer <token>' \
  -H 'Content-Type: application/json' \
  -d '{
    "customer_name": "John Smith",
    "items": [
      {"product_id": "uuid", "name": "Laptop", "quantity": 2, "unit_price": 999.99}
    ]
  }'
```

---

## Products (inventory-service :8003)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/products` | Auth | Create product (unique SKU) |
| `GET` | `/api/v1/products` | Auth | List products with filters |
| `GET` | `/api/v1/products/:id` | Auth | Get product details |
| `PUT` | `/api/v1/products/:id` | Auth | Update product |
| `DELETE` | `/api/v1/products/:id` | Auth | Soft delete product |

**Query parameters**: `sku`, `name`, `category`, `sort_by`, `sort_order`, `limit`, `offset`

---

## Warehouses (inventory-service :8003)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/warehouses` | Auth | Create warehouse |
| `GET` | `/api/v1/warehouses` | Auth | List warehouses |
| `GET` | `/api/v1/warehouses/:id` | Auth | Get warehouse details |
| `PUT` | `/api/v1/warehouses/:id` | Auth | Update warehouse |

**Query parameters**: `name`, `sort_by`, `sort_order`, `limit`, `offset`

---

## Stock (inventory-service :8003)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/v1/stock` | Auth | List stock levels (quantity, reserved, available) |
| `POST` | `/api/v1/stock/reserve` | Auth | Reserve stock (atomic, prevents overselling) |
| `POST` | `/api/v1/stock/release` | Auth | Release previously reserved stock |
| `POST` | `/api/v1/stock/adjust` | Auth | Adjust quantity (inbound/outbound/adjustment) |
| `GET` | `/api/v1/stock/movements` | Auth | Stock movement history log |
| `GET` | `/api/v1/stock/low` | Auth | Products below min_threshold |
| `PUT` | `/api/v1/stock/threshold` | Auth | Set min_threshold per product-warehouse |
| `GET` | `/api/v1/inventory/report` | Auth | Aggregated inventory report |

**Query parameters** (GET /api/v1/stock): `product_id`, `warehouse_id`, `sort_by`, `sort_order`, `limit`, `offset`

**Query parameters** (GET /api/v1/stock/movements): `stock_id`, `product_id`, `warehouse_id`, `type`, `date_from`, `date_to`, `limit`, `offset`

---

## Shipments (logistics-service :8004)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/shipments` | Auth | Create shipment for an order |
| `GET` | `/api/v1/shipments` | Auth | List shipments with filters |
| `GET` | `/api/v1/shipments/:id` | Auth | Get shipment details |
| `PUT` | `/api/v1/shipments/:id/status` | Auth | Update shipment status |
| `POST` | `/api/v1/shipments/bulk-status` | Auth | Bulk status update (partial failure) |

**Query parameters**: `status`, `carrier_id`, `order_id`, `warehouse_id`, `date_from`, `date_to`, `sort_by`, `sort_order`, `limit`, `offset`

**Shipment Status Workflow:**
```
created → picked_up → in_transit → delivered
                                  → failed → returned
```

---

## Carriers (logistics-service :8004)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/carriers` | Auth | Create carrier (ground/air/sea) |
| `GET` | `/api/v1/carriers` | Auth | List carriers |
| `GET` | `/api/v1/carriers/:id` | Auth | Get carrier details |
| `PUT` | `/api/v1/carriers/:id` | Auth | Update carrier |

**Query parameters**: `type`, `is_active`, `name`, `sort_by`, `sort_order`, `limit`, `offset`

---

## Routes (logistics-service :8004)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/routes/calculate` | Auth | Calculate route (mock: distance, duration, cost) |
| `GET` | `/api/v1/logistics/performance` | Auth | Delivery performance (on-time vs late) |

---

## Analytics (analytics-service :8005)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `GET` | `/api/v1/analytics/health` | Auth | Health check |
| `GET` | `/api/v1/analytics/sales` | Auth | Daily sales data |
| `GET` | `/api/v1/analytics/sales/summary` | Auth | Revenue, order count, avg order value |
| `GET` | `/api/v1/analytics/sales/trends` | Auth | Sales trends (day/week granularity) |
| `GET` | `/api/v1/analytics/inventory` | Auth | Daily inventory snapshots |
| `GET` | `/api/v1/analytics/inventory/summary` | Auth | Stock levels, turnover, low-stock count |
| `GET` | `/api/v1/analytics/logistics` | Auth | Daily logistics metrics |
| `GET` | `/api/v1/analytics/logistics/performance` | Auth | Shipment counts, delivery rate, on-time rate |
| `GET` | `/api/v1/analytics/anomalies` | Auth | Rule-based anomaly detection |
| `GET` | `/api/v1/analytics/optimization` | Auth | Reorder recommendations |
| `POST` | `/api/v1/analytics/report` | Auth | Generate custom report (sales/inventory/logistics/full) |

**Common query parameters**: `date_from`, `date_to` (YYYY-MM-DD format)

**Additional** (GET /api/v1/analytics/sales/trends): `granularity` (day or week)

---

## Notifications (notification-service :8006)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `POST` | `/api/v1/notifications` | Admin | Create notification |
| `GET` | `/api/v1/notifications` | Auth | List notifications for current user |
| `PUT` | `/api/v1/notifications/:id/read` | Auth | Mark as read |
| `GET` | `/api/v1/notifications/unread-count` | Auth | Unread notification count |
| `GET` | `/api/v1/notifications/preferences` | Auth | Get notification preferences |
| `PUT` | `/api/v1/notifications/preferences` | Auth | Update preferences per type |
| `POST` | `/api/v1/notifications/bulk` | Admin | Bulk send to multiple users |

**WebSocket**: `ws://localhost:8006/ws/notifications` — real-time notification push

**Notification types**: `order_created`, `order_updated`, `order_cancelled`, `low_stock`, `stock_changed`, `shipment_created`, `shipment_updated`, `system`

---

## MCP Host (mcp-host :8090)

| Method | Endpoint | Auth | Description |
|--------|----------|------|-------------|
| `WS` | `/ws/chat?token=<jwt>` | JWT query param | Chat WebSocket with LLM |
| `GET` | `/health` | Public | Health check with tool count and Redis status |
| `GET` | `/api/v1/mcp/plans/:session_id` | Public | List execution plans for session |
| `GET` | `/api/v1/mcp/plans/:session_id/:plan_id` | Public | Get plan details with steps |

### WebSocket Message Protocol

**Client → Server:**
```json
{"message": "List all orders"}
```

**Server → Client:**
```json
{"type": "thinking", "content": "Processing your request..."}
{"type": "tool_start", "content": "Calling orders_list..."}
{"type": "tool_result", "content": "Retrieved 10 orders"}
{"type": "stream", "content": "Here are "}
{"type": "stream", "content": "the orders..."}
{"type": "message", "content": "Here are the orders:\n\n| # | Customer | Status | Total |\n|..."}
```

**Message types**: `message`, `thinking`, `stream`, `tool_start`, `tool_result`, `tool_error`, `partial_failure`, `system`, `error`

---

## Health Checks

Every service exposes:

| Endpoint | Response |
|----------|----------|
| `GET /health` | `{"status": "ok"}` |
| `GET /health/nats` | `{"status": "ok"}` or `{"status": "disconnected"}` (503) |

---

## Error Responses

```json
{
  "error": {
    "code": "not_found",
    "message": "order not found"
  }
}
```

Common HTTP status codes:
- `400` — Bad Request (validation error)
- `401` — Unauthorized (missing/invalid/expired JWT)
- `403` — Forbidden (insufficient role)
- `404` — Not Found
- `409` — Conflict (duplicate SKU, invalid status transition)
- `429` — Too Many Requests (rate limit: 100 req/min per IP)
- `500` — Internal Server Error
