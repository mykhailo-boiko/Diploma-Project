# Demo Scenarios for Diploma Defense

This document contains 5 scripted MCP chat scenarios demonstrating the core capabilities of ChainOrchestra.
Each scenario includes the role, seed data prerequisites, user inputs, and expected system behavior.

## Prerequisites

```bash
# 1. Start the full stack
docker compose up -d

# 2. Wait for all services to be healthy
docker compose ps  # all should show "healthy"

# 3. Run seed data
./scripts/seed.sh

# 4. Open the frontend
open http://localhost:3000
```

## Test Credentials

| Role              | Email                                   | Password     |
|-------------------|-----------------------------------------|--------------|
| Admin             | admin@chainorchestra.local              | admin123     |
| Operator          | ivan.petrenko@chainorchestra.local        | Operator1!   |
| Warehouse Manager | maria.kovalenko@chainorchestra.local   | Warehouse1!  |
| Logistics Manager | oleksii.shevchenko@chainorchestra.local      | Logistics1!  |
| Analyst           | olena.bondarenko@chainorchestra.local     | Analyst1!    |

---

## Scenario 1: Operator - Order Lifecycle Management

**Role:** Operator (ivan.petrenko@chainorchestra.local / Operator1!)

**Goal:** Demonstrate that an operator can create, view, and manage orders through natural language chat.

### Steps

| # | User Input                                                                 | Expected Behavior                                                                                                    |
|---|---------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------|
| 1 | `Show me all pending orders`                                              | LLM calls `orders_list` with status=pending. Returns a list of pending orders with customer names, totals, dates.    |
| 2 | `Create a new order for customer "Dmytro Ivanenko" with 5 units of "Wireless Mouse" at 29.99 each and 2 units of "USB-C Hub" at 49.99 each` | LLM calls `orders_create`. Returns order details: status=pending, 2 line items, total_amount=249.93.                 |
| 3 | `Show me the details of that order`                                       | Multi-turn context: LLM calls `orders_get` with the order ID from step 2. Returns order with items, status, total.  |
| 4 | `Confirm this order`                                                      | LLM calls `orders_update_status` with status=confirmed. Returns updated order with status=confirmed.                 |
| 5 | `Now show me the order statistics`                                        | LLM calls `orders_stats`. Returns total_orders, total_revenue, breakdown by status (pending, confirmed, etc.).       |
| 6 | `Cancel the order I just created, reason: "Customer changed their mind"`  | LLM calls `orders_cancel` with reason. Returns order with status=cancelled, cancel_reason saved.                     |

### What This Demonstrates
- Natural language to API tool calls
- Multi-turn conversation context (referencing "that order", "this order")
- Full order lifecycle: create -> confirm -> cancel
- Operator has access to order management tools

---

## Scenario 2: Warehouse Manager - Inventory Monitoring

**Role:** Warehouse Manager (maria.kovalenko@chainorchestra.local / Warehouse1!)

**Goal:** Demonstrate inventory visibility, low-stock detection, and stock operations.

### Steps

| # | User Input                                                     | Expected Behavior                                                                                                           |
|---|---------------------------------------------------------------|-----------------------------------------------------------------------------------------------------------------------------|
| 1 | `What items are running low on stock?`                        | LLM calls `stock_low`. Returns products below min_threshold with product name, SKU, current quantity, threshold.            |
| 2 | `Show me the full inventory report`                           | LLM calls `inventory_report`. Returns summary: total products, total quantity, by warehouse breakdown, by category.         |
| 3 | `List all products in the Electronics category`               | LLM calls `products_list` with category=Electronics. Returns list with names, SKUs, prices.                                 |
| 4 | `Show stock levels for the Kyiv warehouse`                  | LLM calls `stock_list` with warehouse filter. Returns stock entries with quantity, reserved, available for each product.     |
| 5 | `What are the recent stock movements?`                        | LLM calls `stock_movements`. Returns movement log: type (inbound/outbound/reserve/release), quantity, timestamps.           |
| 6 | `Show me all warehouses and their status`                     | LLM calls `warehouses_list`. Returns 3 warehouses (Kyiv, Lviv, Odesa) with addresses and active status. |

### What This Demonstrates
- Inventory visibility through natural language
- Low-stock monitoring and alerting
- Cross-entity queries (products, stock, warehouses)
- Warehouse manager has access to inventory tools but not logistics or analytics

---

## Scenario 3: Admin - Multi-Step Workflow

**Role:** Admin (admin@chainorchestra.local / admin123)

**Goal:** Demonstrate complex multi-step workflow: create order, check inventory, create shipment, send notification.

### Steps

| # | User Input                                                                                          | Expected Behavior                                                                                              |
|---|-----------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------|
| 1 | `Create an order for customer "Nataliia Smiian" with 10 units of "Standing Desk" at 599.99 each`  | LLM calls `orders_create`. Returns order with status=pending, total=5999.90.                                   |
| 2 | `Confirm this order and check if we have enough stock of Standing Desks`                            | Multi-step: LLM calls `orders_update_status` (confirmed) + `stock_list` or `products_list`. Reports both results. |
| 3 | `Show me available carriers for this shipment`                                                      | LLM calls `carriers_list`. Returns 3 carriers with types and cost_per_km.                                      |
| 4 | `Calculate a delivery route from Kyiv to Odesa using the ground carrier`                    | LLM calls `routes_calculate` with origin, destination, carrier_id. Returns distance, duration, cost.           |
| 5 | `Send a notification to user ivan.petrenko@chainorchestra.local that the order has been confirmed`    | LLM calls `notifications_create`. Returns created notification with type and content.                          |
| 6 | `Give me a summary: how many orders do we have and what's the overall revenue?`                     | LLM calls `orders_stats`. Returns aggregated statistics across all orders.                                     |

### What This Demonstrates
- Admin has unrestricted access to ALL tools across all services
- Multi-step workflows spanning multiple services (orders + inventory + logistics + notifications)
- LLM orchestrates multiple tool calls in a single conversation turn
- Route calculation with realistic cost estimation
- Cross-service data correlation

---

## Scenario 4: Analyst - Reports and Anomaly Detection

**Role:** Analyst (olena.bondarenko@chainorchestra.local / Analyst1!)

**Goal:** Demonstrate analytics capabilities: sales trends, anomaly detection, optimization recommendations.

### Steps

| # | User Input                                                          | Expected Behavior                                                                                                  |
|---|--------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------|
| 1 | `Show me the sales summary for the last 30 days`                   | LLM calls `analytics_sales_summary` with date range. Returns revenue, order_count, avg_order_value.                |
| 2 | `What are the sales trends over the past month?`                   | LLM calls `analytics_sales_trends` with granularity=day. Returns daily trend data (revenue, order count per day).  |
| 3 | `Are there any anomalies in our data?`                             | LLM calls `analytics_anomalies`. Returns detected anomalies (revenue spikes, zero-order days, high failure rates). |
| 4 | `What optimization recommendations do you have for our inventory?` | LLM calls `analytics_optimization`. Returns reorder recommendations: product, reorder_point, recommended_qty.      |
| 5 | `Generate a full report for management`                            | LLM calls `analytics_report` with type=full. Returns combined sales + inventory + logistics report as JSON.        |
| 6 | `How is our logistics performance?`                                | LLM calls `analytics_logistics_performance`. Returns on-time rate, delivery stats, avg delivery hours.             |

### What This Demonstrates
- Analytics-only access for analyst role
- Statistical analysis through natural language
- Anomaly detection (rule-based, threshold-driven)
- Inventory optimization recommendations (demand-based reorder points)
- Report generation capability

---

## Scenario 5: RBAC Demo - Access Control Enforcement

**Role:** Operator first, then Admin (to compare)

**Goal:** Demonstrate that RBAC prevents unauthorized access through the MCP chat interface.

### Part A: Operator (ivan.petrenko@chainorchestra.local / Operator1!)

| # | User Input                                      | Expected Behavior                                                                                      |
|---|-------------------------------------------------|--------------------------------------------------------------------------------------------------------|
| 1 | `Show me all orders`                            | SUCCESS: LLM calls `orders_list`. Returns order data. Operator HAS orders access.                      |
| 2 | `Show me the inventory report`                  | DENIED: LLM responds that inventory capabilities are not available for this role. No `inventory_*` tools visible. |
| 3 | `List all shipments`                            | DENIED: LLM responds that logistics/shipment capabilities are not available. No `shipments_*` tools visible.      |
| 4 | `Generate an analytics report`                  | DENIED: LLM responds that analytics capabilities are not available. No `analytics_*` tools visible.               |
| 5 | `Show me all users in the system`               | DENIED: LLM responds that user management is not available. No `users_list` tool visible.                         |
| 6 | `Show me my unread notifications`               | SUCCESS: LLM calls `notifications_unread_count`. Returns count. Operator HAS notifications access.     |

### Part B: Admin (admin@chainorchestra.local / admin123)

| # | User Input                                      | Expected Behavior                                                                                    |
|---|-------------------------------------------------|------------------------------------------------------------------------------------------------------|
| 1 | `Show me the inventory report`                  | SUCCESS: LLM calls `inventory_report`. Returns full inventory data. Admin has ALL access.            |
| 2 | `List all shipments`                            | SUCCESS: LLM calls `shipments_list`. Returns shipment data.                                          |
| 3 | `Generate a full analytics report`              | SUCCESS: LLM calls `analytics_report` with type=full. Returns comprehensive report.                  |
| 4 | `List all users in the system`                  | SUCCESS: LLM calls `users_list`. Returns all 5 users with roles.                                    |

### What This Demonstrates
- RBAC enforcement at the MCP tool level (tools filtered BEFORE sending to LLM)
- Operator can only access orders + notifications
- Admin has unrestricted access to all 93 tools
- Same natural language queries produce different results based on role
- Security boundary is enforced at the orchestration layer, not just the UI

---

## RBAC Access Matrix (Quick Reference)

| Capability        | Admin | Operator | Warehouse Mgr | Logistics Mgr | Analyst |
|-------------------|:-----:|:--------:|:--------------:|:--------------:|:-------:|
| Orders            |  Yes  |   Yes    |      Yes       |      Yes       |   No    |
| Products/Stock    |  Yes  |    No    |      Yes       |       No       |   No    |
| Warehouses        |  Yes  |    No    |      Yes       |       No       |   No    |
| Shipments/Carriers|  Yes  |    No    |       No       |      Yes       |   No    |
| Routes            |  Yes  |    No    |       No       |      Yes       |   No    |
| Analytics         |  Yes  |    No    |       No       |       No       |  Yes    |
| Notifications     |  Yes  |   Yes    |       No       |       No       |   No    |
| User Management   |  Yes  |    No    |       No       |       No       |   No    |
| Profile/Auth      |  Yes  |   Yes    |      Yes       |      Yes       |  Yes    |

---

## Tips for the Defense Presentation

1. **Start with Scenario 1** (Operator) — shows basic chat-to-API flow
2. **Then Scenario 5** (RBAC) — demonstrates security enforcement
3. **Then Scenario 3** (Admin multi-step) — shows the most impressive multi-service orchestration
4. **Scenario 4** (Analyst) — demonstrates analytics intelligence
5. **Keep Scenario 2** as backup if time permits

### Presentation Flow

```
Login as Operator → Create/manage order (Scenario 1)
    → Try accessing inventory → DENIED (Scenario 5A)
    → Switch to Admin → Full access (Scenario 5B)
    → Multi-step workflow (Scenario 3)
    → Switch to Analyst → Reports and anomalies (Scenario 4)
```

### Timing

- Each scenario: ~3-5 minutes
- Total demo time: ~15-20 minutes
- Allow 2-3 seconds between messages for LLM processing
