# PRD: Intelligent Supply Chain & Logistics Orchestration Platform

## Metadata

| Field            | Value                                           |
| ---------------- | ----------------------------------------------- |
| **Project Name** | ChainOrchestra — AI-Powered Supply Chain Platform |
| **Version**      | 1.0 (MVP)                                       |
| **Date**         | 2026-04-11                                       |
| **Deadline**     | 2026-04-25 (2 weeks)                             |
| **Type**         | Diploma Project — Enterprise Platform            |
| **Platform**     | Web (responsive, no native mobile)               |

---

## 1. Product Overview

### 1.1 Vision

Enterprise-grade supply chain management platform with an AI orchestration layer built on MCP (Model Context Protocol). The platform combines traditional UI-driven workflows with a natural language interface that translates user **intents** into automated multi-step **workflows** across microservices.

### 1.2 Problem Statement

| Problem                                   | Impact                                                  |
| ----------------------------------------- | ------------------------------------------------------- |
| Too many disconnected interfaces          | High cognitive load, context switching, human errors     |
| Gap between insights and actions          | Data exists but acting on it requires 10+ manual steps   |
| Weak automation of complex processes      | Only simple CRUD is automated; multi-step flows are manual |
| No universal management interface         | Each module lives separately with its own UI              |

### 1.3 Solution

A **closed-world simulation** of a supply chain environment where:

- All business domains (orders, inventory, logistics, analytics, notifications) are managed through **microservices**
- An **MCP orchestrator** provides a natural language chat interface that understands user intent, builds execution plans (DAGs), and orchestrates microservice calls
- The MCP layer respects **RBAC** — each user role can only trigger actions within their permission scope
- All external integrations use an **adapter pattern** with mock implementations for MVP

### 1.4 Key Differentiator

> Users manage the business through **intents**, not interfaces. A single natural language request replaces 10–20 minutes of clicking across multiple screens.

---

## 2. Target Audience

### 2.1 User Roles

| Role                  | Primary Responsibilities                            | MCP Scope                                         |
| --------------------- | --------------------------------------------------- | ------------------------------------------------- |
| **Admin**             | Full system access, user management, configuration   | Unrestricted — all tools and services              |
| **Warehouse Manager** | Inventory management, stock levels, warehouse ops    | Inventory tools, warehouse reports, stock alerts    |
| **Logistics Manager** | Delivery management, routing, carrier coordination   | Logistics tools, shipment tracking, route reports   |
| **Analyst**           | Reports, dashboards, data analysis, forecasting      | Analytics tools, read-only data access, exports     |
| **Operator**          | Order processing, status updates, customer handling  | Order tools, basic inventory queries                |

### 2.2 Authentication

- **Method:** Email/password (JWT token-based sessions)
- **Session:** Persistent across browser restarts via refresh tokens
- **Password recovery:** Email-based reset flow (mock email adapter in MVP)

### 2.3 Authorization Model

- Role-Based Access Control (RBAC)
- Roles assigned at user creation by Admin
- Each API endpoint and MCP tool is tagged with required roles
- MCP orchestrator validates user role before executing any tool

---

## 3. Core Features & Functionality

### 3.1 Order Management Service

**Owner:** `order-service` (Go)

#### Entities

| Entity       | Key Fields                                                                                           |
| ------------ | ---------------------------------------------------------------------------------------------------- |
| **Order**    | `id` (uuid), `customer_name`, `customer_email`, `status` (enum), `items` (relation), `total_amount` (decimal), `created_at`, `updated_at`, `deleted_at` |
| **OrderItem**| `id` (uuid), `order_id` (fk), `product_id` (fk), `quantity` (int), `unit_price` (decimal), `total_price` (decimal) |

#### Order Statuses

`pending` → `confirmed` → `processing` → `shipped` → `delivered` → `completed`

Side transitions: any status → `cancelled`, `shipped` → `returned`

#### Features

| Feature               | Description                                                          | Acceptance Criteria                                                                                         |
| --------------------- | -------------------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------- |
| Create Order          | Create order with line items, validate stock availability             | Order created with status `pending`; inventory reserved; event `order.created` published to NATS              |
| View Orders           | List with filters (status, date range, customer), pagination, sorting | Supports `Filter`, `Sort`, `Page` pattern; response < 200ms for 1000 records                                 |
| Order Details         | Full order with items, status history, related shipment               | All relations loaded; status timeline displayed                                                               |
| Update Order Status   | Transition order through workflow states                              | Only valid transitions allowed; event `order.status_changed` published; inventory updated on cancellation      |
| Cancel Order          | Cancel with reason; release reserved inventory                        | Inventory released; notification sent; event published                                                        |
| Order Search          | Full-text search by customer name, order ID                           | Results returned within 300ms; minimum 2 characters                                                           |

#### MCP Tools

```
tool: orders.create       — Create a new order
tool: orders.list         — List/filter orders
tool: orders.get          — Get order details by ID
tool: orders.updateStatus — Transition order status
tool: orders.cancel       — Cancel an order
tool: orders.search       — Search orders by keyword
tool: orders.stats        — Get order statistics (count by status, revenue)
```

---

### 3.2 Inventory Management Service

**Owner:** `inventory-service` (Go)

#### Entities

| Entity          | Key Fields                                                                                           |
| --------------- | ---------------------------------------------------------------------------------------------------- |
| **Product**     | `id` (uuid), `sku` (string, unique), `name`, `description`, `category`, `unit_price` (decimal), `created_at`, `updated_at`, `deleted_at` |
| **Stock**       | `id` (uuid), `product_id` (fk), `warehouse_id` (fk), `quantity` (int), `reserved` (int), `available` (computed: quantity - reserved), `min_threshold` (int), `updated_at` |
| **Warehouse**   | `id` (uuid), `name`, `location`, `capacity` (int), `current_load` (int), `status` (enum: active/inactive), `created_at` |
| **StockMovement** | `id` (uuid), `product_id`, `warehouse_id`, `type` (enum: inbound/outbound/transfer/adjustment), `quantity` (int), `reference_id`, `created_at` |

#### Features

| Feature                  | Description                                                  | Acceptance Criteria                                                                              |
| ------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| Product CRUD             | Create, read, update, soft-delete products                    | SKU uniqueness enforced; soft delete via `deleted_at`                                             |
| Stock Management         | Track quantity per product per warehouse                       | Real-time available = quantity - reserved; event on stock change                                   |
| Stock Reservation        | Reserve stock for confirmed orders                            | Atomic operation; prevents overselling; released on order cancellation                             |
| Low Stock Alerts         | Trigger when available < min_threshold                        | Event `inventory.low_stock` published; visible in dashboard                                        |
| Stock Movements Log      | Audit trail for all stock changes                             | Every change recorded with type, quantity, reference                                                |
| Warehouse Management     | CRUD for warehouses, capacity tracking                        | Current load updated on stock movements; capacity overflow prevented                               |
| Inventory Report         | Summary by warehouse, by category, low stock items            | Aggregated data; exportable                                                                        |

#### MCP Tools

```
tool: inventory.products.list      — List/filter products
tool: inventory.products.get       — Get product details
tool: inventory.products.create    — Create product
tool: inventory.products.update    — Update product
tool: inventory.stock.get          — Get stock levels for product/warehouse
tool: inventory.stock.adjust       — Adjust stock quantity
tool: inventory.stock.lowStock     — Get all low-stock items
tool: inventory.warehouses.list    — List warehouses
tool: inventory.warehouses.stats   — Warehouse utilization stats
tool: inventory.movements.list     — Stock movement history
tool: inventory.report             — Generate inventory summary report
```

---

### 3.3 Logistics & Delivery Service

**Owner:** `logistics-service` (Go)

#### Entities

| Entity        | Key Fields                                                                                             |
| ------------- | ------------------------------------------------------------------------------------------------------ |
| **Shipment**  | `id` (uuid), `order_id` (fk), `warehouse_id` (fk), `carrier` (string), `tracking_number`, `status` (enum), `estimated_delivery`, `actual_delivery`, `created_at`, `updated_at` |
| **Route**     | `id` (uuid), `shipment_id` (fk), `origin`, `destination`, `distance_km` (decimal), `estimated_duration_min` (int), `status` (enum) |
| **Carrier**   | `id` (uuid), `name`, `type` (enum: ground/air/sea), `cost_per_km` (decimal), `max_weight_kg` (decimal), `status` (enum: active/inactive) |

#### Shipment Statuses

`created` → `picked_up` → `in_transit` → `out_for_delivery` → `delivered`

Side: any → `failed`, `returned`

#### Features

| Feature                  | Description                                            | Acceptance Criteria                                                                        |
| ------------------------ | ------------------------------------------------------ | ------------------------------------------------------------------------------------------ |
| Create Shipment          | Auto-create from confirmed order; assign carrier        | Shipment linked to order; carrier selected by cost/availability; event published             |
| Shipment Tracking        | Status updates through lifecycle                        | Each status change timestamped; event `logistics.shipment.status_changed` published          |
| Route Simulation         | Mock route calculation (distance, duration, cost)       | Deterministic mock based on origin/destination strings; no external API                      |
| Carrier Management       | CRUD for carriers with pricing                          | Cost calculation: `distance_km * cost_per_km`; carrier availability check                    |
| Delivery Performance     | Track on-time vs late deliveries                        | Computed from `estimated_delivery` vs `actual_delivery`                                      |
| Bulk Shipment Status     | Update multiple shipments at once                       | Batch operation; partial failure handling; individual events per shipment                     |

#### MCP Tools

```
tool: logistics.shipments.create   — Create shipment for order
tool: logistics.shipments.list     — List/filter shipments
tool: logistics.shipments.get      — Get shipment details
tool: logistics.shipments.track    — Get current shipment status
tool: logistics.shipments.update   — Update shipment status
tool: logistics.routes.calculate   — Simulate route (mock)
tool: logistics.carriers.list      — List available carriers
tool: logistics.performance        — Delivery performance report
```

---

### 3.4 Analytics & Reporting Service

**Owner:** `analytics-service` (Go)

#### Data Sources

Consumes events from all other services via NATS. Stores pre-aggregated data in dedicated PostgreSQL tables (or ClickHouse if implemented).

#### Features

| Feature                  | Description                                                  | Acceptance Criteria                                                                              |
| ------------------------ | ------------------------------------------------------------ | ------------------------------------------------------------------------------------------------ |
| Sales Dashboard          | Revenue, order count, avg order value, trends over time       | Real-time from events; filterable by date range                                                   |
| Inventory Dashboard      | Stock levels, turnover rate, low stock items, warehouse load  | Updated on stock events; visual charts                                                            |
| Logistics Dashboard      | Shipment status distribution, on-time rate, carrier perf      | Updated on shipment events; filterable by carrier/period                                          |
| Custom Report Generation | Build reports from available metrics with date range filters  | MCP can generate reports via natural language request                                              |
| Anomaly Detection        | Identify unusual patterns (demand spikes, delivery delays)    | Rule-based detection (threshold-based); flagged in dashboard and via notifications                 |
| Supply Optimization      | Suggest reorder quantities based on historical demand          | Simple algorithm: avg daily demand * lead time + safety stock                                     |
| Data Export              | Export reports as JSON                                        | Endpoint returns structured JSON; MCP tool available                                              |

#### MCP Tools

```
tool: analytics.sales.summary         — Sales metrics for period
tool: analytics.sales.trends          — Revenue/order trends over time
tool: analytics.inventory.summary     — Inventory health overview
tool: analytics.logistics.performance — Delivery/carrier performance
tool: analytics.report.generate       — Generate custom report by parameters
tool: analytics.anomalies.detect      — Run anomaly detection on recent data
tool: analytics.optimization.suggest  — Supply optimization recommendations
```

---

### 3.5 Notification Service

**Owner:** `notification-service` (Go)

#### Entities

| Entity           | Key Fields                                                                                 |
| ---------------- | ------------------------------------------------------------------------------------------ |
| **Notification** | `id` (uuid), `user_id`, `type` (enum), `channel` (enum: in_app/email/sms), `title`, `body`, `status` (enum: pending/sent/read/failed), `metadata` (jsonb), `created_at`, `read_at` |

#### Notification Types

- `order_status_changed` — Order moved to new status
- `low_stock_alert` — Product below threshold
- `shipment_update` — Shipment status changed
- `anomaly_detected` — Analytics found anomaly
- `system_alert` — System-level notifications
- `mcp_task_completed` — MCP workflow finished

#### Features

| Feature               | Description                                         | Acceptance Criteria                                                     |
| --------------------- | --------------------------------------------------- | ----------------------------------------------------------------------- |
| In-App Notifications  | Real-time notifications in UI via WebSocket          | Delivered < 1s after event; badge counter in header                      |
| Email Notifications   | Mock email adapter (logs to console/DB)              | Adapter interface; mock implementation logs email content                 |
| SMS Notifications     | Mock SMS adapter                                     | Same adapter pattern as email                                            |
| Notification History  | User can view past notifications                     | Paginated; filterable by type; mark as read                              |
| Notification Prefs    | User can toggle notification types on/off            | Per-user settings; respected before sending                              |
| Bulk Notifications    | Admin can send system-wide notifications             | Role: Admin only; delivered to all active users                          |

#### MCP Tools

```
tool: notifications.list       — List user notifications
tool: notifications.markRead   — Mark notification as read
tool: notifications.send       — Send notification (admin)
tool: notifications.prefs.get  — Get user notification preferences
tool: notifications.prefs.set  — Update notification preferences
```

---

### 3.6 User Management Service

**Owner:** `user-service` (Go)

#### Entities

| Entity    | Key Fields                                                                                         |
| --------- | -------------------------------------------------------------------------------------------------- |
| **User**  | `id` (uuid), `email` (unique), `password_hash`, `name`, `role` (enum), `status` (enum: active/inactive), `last_login_at`, `created_at`, `updated_at`, `deleted_at` |

#### Features

| Feature           | Description                              | Acceptance Criteria                                                     |
| ----------------- | ---------------------------------------- | ----------------------------------------------------------------------- |
| Registration      | Admin creates users with role assignment  | Email uniqueness; password hashed with bcrypt; event published           |
| Login             | Email/password → JWT (access + refresh)   | Access token: 15min; refresh token: 7days; stored in httpOnly cookie     |
| Profile           | User can view/edit own profile            | Cannot change own role                                                   |
| User Management   | Admin CRUD on users                       | Admin only; soft delete; role reassignment                               |
| Password Reset    | Request → mock email → reset              | Token-based; 1hr expiry; mock email adapter                              |

#### MCP Tools

```
tool: users.list        — List users (admin)
tool: users.get         — Get user profile
tool: users.create      — Create user (admin)
tool: users.update      — Update user (admin)
tool: users.deactivate  — Deactivate user (admin)
```

---

### 3.7 MCP Orchestrator (AI Layer)

**Owner:** `mcp-orchestrator` (Python)

#### MCP Protocol Overview

MCP (Model Context Protocol) is an open JSON-RPC 2.0 based protocol (spec version `2025-06-18`) that standardizes how AI applications connect to external tools and data. Architecture has three participants:

- **Host** — the AI application that manages MCP client connections and LLM interaction
- **Client** — maintains a stateful session per server; routes JSON-RPC messages
- **Server** — exposes capabilities (tools, resources, prompts) via the protocol

In this platform, the **MCP Host** (Python) connects to a **single MCP Server** (Python) that wraps all Go microservice endpoints as MCP tools. This is the pragmatic approach for the 2-week timeline — a single MCP server acts as a bridge/adapter to all Go REST services.

#### Architecture

```
Frontend (WebSocket)
    ↓
MCP Host (Python — AI Agent)
  ├── LLM (Google Gemini API) — intent parsing, planning, response formatting
  ├── Context Manager (Redis) — conversation history, user role
  └── MCP Client
        ↓ (JSON-RPC 2.0 over stdio)
      MCP Server (Python — FastMCP)
        ├── Tool: orders.*        → HTTP calls to order-service
        ├── Tool: inventory.*     → HTTP calls to inventory-service
        ├── Tool: logistics.*     → HTTP calls to logistics-service
        ├── Tool: analytics.*     → HTTP calls to analytics-service
        ├── Tool: notifications.* → HTTP calls to notification-service
        └── Tool: users.*         → HTTP calls to user-service
```

#### Components

| Component          | Responsibility                                                                |
| ------------------ | ----------------------------------------------------------------------------- |
| **MCP Host**       | AI agent: manages LLM, MCP client, WebSocket endpoint for chat, auth/role extraction |
| **MCP Server**     | FastMCP-based server exposing all Go service endpoints as MCP tools             |
| **Tool Registry**  | Auto-generated from Python type hints and docstrings via FastMCP                |
| **Planner**        | LLM-driven: system prompt instructs LLM to decompose complex requests into multi-step tool calls |
| **Context Manager**| Redis-based conversation context for multi-turn dialogues                       |
| **RBAC Filter**    | Filters available tools based on user role before sending tool list to LLM       |

#### MCP Tool Implementation (FastMCP Pattern)

Tools are defined using Python FastMCP SDK with type hints. The SDK auto-generates JSON Schema `inputSchema` from type annotations:

```python
from mcp.server.fastmcp import FastMCP

mcp = FastMCP("supply-chain-tools")

@mcp.tool()
async def orders_list(
    status: str | None = None,
    date_from: str | None = None,
    date_to: str | None = None,
    customer_name: str | None = None,
    limit: int = 20,
    offset: int = 0,
) -> str:
    """List orders with optional filters by status, date range, customer.

    Args:
        status: Order status filter (pending, confirmed, processing, shipped, delivered, completed, cancelled)
        date_from: Start date filter (YYYY-MM-DD)
        date_to: End date filter (YYYY-MM-DD)
        customer_name: Customer name search
        limit: Max results per page
        offset: Pagination offset
    """
    # HTTP call to order-service
    response = await http_client.get(
        "http://order-service:8002/api/v1/orders",
        params={"status": status, "date_from": date_from, ...}
    )
    return response.json()
```

This generates the MCP-compliant tool definition:

```json
{
  "name": "orders_list",
  "description": "List orders with optional filters by status, date range, customer.",
  "inputSchema": {
    "type": "object",
    "properties": {
      "status": { "type": "string", "nullable": true },
      "date_from": { "type": "string", "nullable": true },
      "date_to": { "type": "string", "nullable": true },
      "customer_name": { "type": "string", "nullable": true },
      "limit": { "type": "integer", "default": 20 },
      "offset": { "type": "integer", "default": 0 }
    }
  }
}
```

#### RBAC in MCP

The MCP Host filters tools before sending them to the LLM:

```python
# Role-to-tools mapping
ROLE_PERMISSIONS = {
    "admin": ["*"],  # all tools
    "operator": ["orders_*", "inventory_stock_get", "notifications_*"],
    "warehouse_manager": ["inventory_*", "notifications_*"],
    "logistics_manager": ["logistics_*", "notifications_*"],
    "analyst": ["analytics_*", "orders_list", "inventory_stock_get", "logistics_shipments_list"],
}
```

The LLM only sees tools the user's role permits. If a user asks for something outside their scope, the LLM naturally responds that the capability is unavailable (it doesn't see the tool).

#### Execution Flow Example

**User input:** "Show me all late deliveries this week and notify the logistics manager"

The LLM (with tool-use capability) autonomously decides the execution plan:

```
LLM Call 1 → tool_use: logistics_shipments_list(status="in_transit", overdue=true)
  ← result: [3 late shipments]

LLM Call 2 → tool_use (parallel):
  ├── analytics_logistics_performance(date_range="this_week")
  └── users_list(role="logistics_manager")
  ← results: [performance data], [logistics manager user]

LLM Call 3 → tool_use: notifications_send(user_id="...", title="Late deliveries alert", body="...")
  ← result: notification sent

LLM Call 4 → final text response summarizing all actions taken
```

The system supports two execution modes:

1. **LLM-driven tool chaining** (default for MVP simplicity) — the LLM handles planning natively via its tool-use capability. The system prompt instructs the LLM to decompose complex requests into sequential tool calls.
2. **Structured workflow execution via DAG** (extensible architecture) — predefined workflow templates can be executed deterministically without LLM involvement, enabling repeatable multi-step operations.

The MCP Orchestrator retains control over execution, validation, and error handling, ensuring deterministic behavior even when using LLM-generated plans.

#### Features

| Feature                | Description                                             | Acceptance Criteria                                                         |
| ---------------------- | ------------------------------------------------------- | --------------------------------------------------------------------------- |
| Natural Language Chat  | WebSocket-based conversational interface                  | Real-time responses; conversation history maintained                          |
| MCP Tool Discovery     | `tools/list` returns all available tools for user's role   | Tools filtered by RBAC; JSON Schema validated                                 |
| MCP Tool Execution     | `tools/call` invokes Go services via HTTP bridge           | Results returned as structured content; errors handled gracefully              |
| Multi-Step Workflows   | LLM decomposes complex requests into sequential tool calls | LLM autonomously chains tool calls; handles dependencies between steps         |
| RBAC Enforcement       | Tool list filtered by user role before LLM sees it         | Users cannot invoke tools outside their role; clean "not available" responses   |
| Conversation Context   | Multi-turn context maintained in Redis                     | "Show me more details" refers to previous result; 30min session TTL            |
| Error Handling         | Graceful handling of tool failures                         | Partial results returned; failed steps reported; no system crash               |
| Streaming Responses    | SSE-based progressive output as LLM generates              | User sees tokens as they arrive, not just final result                         |

#### LLM Integration

- **Provider:** Google Gemini API (Gemini 2.5 Flash or latest available)
- **Usage:** Function calling mode — LLM receives system instruction + tool declarations + conversation history
- **System instruction:** Contains platform context, user role, available actions description
- **Temperature:** 0.1 for tool selection accuracy
- **Fallback:** If LLM cannot parse intent → ask clarifying question (not an error)

#### MCP Architectural Principles

- **Separation of concerns** between planning (LLM) and execution (orchestrator)
- **Tool-based abstraction** of all backend capabilities
- **Deterministic execution layer** independent of LLM
- **Role-aware capability exposure** (RBAC at tool level)
- **Extensibility** for workflow templates and DAG-based execution

#### MCP vs LLM Boundary

The MCP Orchestrator is not equivalent to the LLM. The LLM is used only for intent understanding and planning, while execution, validation, and orchestration remain fully controlled by the MCP layer. This separation ensures that business logic, access control, and error handling are never delegated to a non-deterministic component.

#### Execution Guarantees

| Guarantee                     | Description                                                   |
| ----------------------------- | ------------------------------------------------------------- |
| Idempotent tool execution     | Safe retries — repeated calls produce the same result         |
| Partial failure handling      | Completed steps preserved; failed steps reported to user      |
| Timeout handling per tool     | Each tool call has a configurable execution timeout           |
| Retry policy                  | Transient failures (network, 5xx) retried with backoff        |
| No silent failures            | All errors surfaced to user with context                      |

#### Why MCP over Traditional API

| Aspect            | Traditional approach                                  | MCP approach                                            |
| ----------------- | ----------------------------------------------------- | ------------------------------------------------------- |
| User interaction  | User → UI → multiple manual API calls                 | User → Intent → MCP → automated orchestration → result  |
| Integration cost  | Each new feature requires frontend + backend changes  | New tool = new MCP function, immediately available via chat |
| Multi-step tasks  | User manually sequences operations                    | LLM decomposes intent into tool chain automatically     |
| Discoverability   | User must know which screen/button to use             | User describes what they need in natural language        |

---

## 4. Technical Stack

### 4.1 Summary

| Layer          | Technology                                    |
| -------------- | --------------------------------------------- |
| **Backend**    | Go 1.23+ (chi router)                         |
| **AI Layer**   | Python 3.12+ (FastMCP SDK + Google Gemini SDK)   |
| **Frontend**   | Next.js 15+ / TypeScript / TailwindCSS         |
| **Primary DB** | PostgreSQL 16                                  |
| **Cache**      | Redis 7                                        |
| **Analytics**  | PostgreSQL (separate schema; ClickHouse optional) |
| **Messaging**  | NATS                                           |
| **LLM**        | Google Gemini API (Gemini 2.5 Flash)             |
| **MCP SDK**    | Python FastMCP (mcp package)                    |
| **Infra**      | Docker + Docker Compose                         |
| **Observability** | Prometheus + Grafana + OpenTelemetry          |

### 4.2 Backend (Go) Details

- **Router:** chi (lightweight, stdlib-compatible)
- **Architecture:** Clean Architecture per service (handler → service → storage)
- **Database driver:** pgx (PostgreSQL)
- **Migrations:** golang-migrate
- **Logging:** zap (structured)
- **Config:** environment variables (envconfig or go-flags)
- **Validation:** go-playground/validator
- **API format:** REST (JSON), consistent error format

### 4.3 Frontend Details

- **Framework:** Next.js 15+ (App Router)
- **Language:** TypeScript (strict mode)
- **Styling:** TailwindCSS
- **Data fetching:** TanStack Query (React Query)
- **Charts:** Recharts or ECharts
- **WebSocket:** native WebSocket API for MCP chat and notifications
- **State management:** Zustand (lightweight, minimal boilerplate)
- **Form handling:** React Hook Form + Zod validation
- **Routing:** Next.js App Router (file-based)

### 4.4 Inter-Service Communication

| Type          | Technology | Usage                                               |
| ------------- | ---------- | --------------------------------------------------- |
| Synchronous   | HTTP/REST  | MCP orchestrator → Go services; Frontend → services   |
| Asynchronous  | NATS       | Domain events (order.created, stock.changed, etc.)    |
| Real-time     | WebSocket  | MCP chat, in-app notifications                        |

### 4.5 API Gateway

A single **API Gateway** (Go, chi-based) sits in front of all services:

- JWT validation
- Role extraction and forwarding (X-User-ID, X-User-Role headers)
- Rate limiting
- Request routing to appropriate service
- CORS handling

---

## 5. Conceptual Data Model

### 5.1 Service-to-Database Mapping

Each service owns its database schema (database-per-service pattern):

```
user-service       → PostgreSQL: schema "users"
order-service      → PostgreSQL: schema "orders"
inventory-service  → PostgreSQL: schema "inventory"
logistics-service  → PostgreSQL: schema "logistics"
analytics-service  → PostgreSQL: schema "analytics"
notification-service → PostgreSQL: schema "notifications"
```

### 5.2 Entity Relationships (Cross-Service)

```
User (user-service)
  └── creates → Order (order-service)
                  ├── contains → OrderItem (order-service)
                  │                 └── references → Product.id (inventory-service) [eventual consistency]
                  └── triggers → Shipment (logistics-service)
                                   └── has → Route (logistics-service)

Product (inventory-service)
  └── tracked in → Stock (inventory-service)
                     └── located at → Warehouse (inventory-service)
                     └── logged via → StockMovement (inventory-service)

All events → consumed by → analytics-service (pre-aggregated tables)
All events → consumed by → notification-service (notification rules)
```

### 5.3 Cross-Service Data Consistency

- **Pattern:** Eventual consistency via NATS events
- **No cross-service JOINs** — services communicate via HTTP for queries, NATS for events
- **ID references:** Services store foreign IDs (e.g., `product_id` in OrderItem) but do not enforce FK constraints across services

---

## 6. UI Design Principles

### 6.1 Layout

- **Sidebar navigation** — collapsible, role-aware (shows only relevant sections)
- **Top bar** — user profile, notification bell with badge, MCP chat toggle
- **Content area** — responsive grid, card-based dashboard widgets
- **MCP Chat panel** — right-side sliding panel, always accessible

### 6.2 Core Pages

| Page                  | Roles                         | Description                                    |
| --------------------- | ----------------------------- | ---------------------------------------------- |
| Dashboard             | All                           | Role-specific KPI widgets and quick actions      |
| Orders List           | Admin, Operator                | Table with filters, sorting, pagination          |
| Order Details         | Admin, Operator                | Full order view with timeline and actions         |
| Products              | Admin, Warehouse Manager       | Product catalog management                       |
| Inventory / Stock     | Admin, Warehouse Manager       | Stock levels by warehouse, movements log          |
| Warehouses            | Admin, Warehouse Manager       | Warehouse list, capacity visualization            |
| Shipments             | Admin, Logistics Manager       | Shipment tracking table and status management     |
| Carriers              | Admin, Logistics Manager       | Carrier management                               |
| Analytics             | Admin, Analyst                 | Charts, reports, anomaly alerts                   |
| Notifications         | All                           | Notification history and preferences              |
| User Management       | Admin                          | User CRUD                                        |
| Settings              | Admin                          | System configuration                             |
| MCP Chat              | All (role-scoped)              | Natural language interface panel                  |

### 6.3 Design System

- **Color palette:** Professional blues/grays; status colors (green=success, yellow=warning, red=error, blue=info)
- **Typography:** Inter or system fonts; clear hierarchy (h1–h4, body, caption)
- **Components:** Consistent card, table, form, modal, toast patterns
- **Responsive:** Desktop-first, functional on tablet; minimum viewport: 768px
- **Dark mode:** Not in MVP scope

---

## 7. Security Considerations

| Area                    | Implementation                                                        |
| ----------------------- | --------------------------------------------------------------------- |
| Authentication          | JWT (access 15min + refresh 7d); bcrypt password hashing               |
| Authorization           | RBAC enforced at API Gateway + MCP orchestrator                        |
| Input Validation        | Server-side validation on every endpoint; parameterized SQL queries     |
| API Security            | Rate limiting at gateway; CORS whitelist; HTTPS (in production)         |
| Data Protection         | Soft delete (no permanent deletion in MVP); audit trail on mutations    |
| MCP Security            | Role validated before each tool execution; no raw SQL or shell tools    |
| Secrets Management      | Environment variables; no secrets in code or Docker images              |
| Dependency Security     | Minimal dependencies; known-good versions pinned in go.mod/requirements |

---

## 8. Event-Driven Architecture

### 8.1 NATS Subjects

```
events.orders.created
events.orders.status_changed
events.orders.cancelled
events.inventory.stock_changed
events.inventory.low_stock
events.logistics.shipment_created
events.logistics.shipment_status_changed
events.analytics.anomaly_detected
events.notifications.send
events.mcp.task_completed
```

### 8.2 Event Schema (Standard Envelope)

```json
{
  "id": "uuid",
  "type": "orders.status_changed",
  "source": "order-service",
  "timestamp": "2026-04-11T12:00:00Z",
  "data": {
    "order_id": "uuid",
    "old_status": "confirmed",
    "new_status": "processing",
    "changed_by": "user-uuid"
  }
}
```

### 8.3 Consumer Matrix

| Event                                | Consumers                           |
| ------------------------------------ | ----------------------------------- |
| `events.orders.created`              | analytics, notifications             |
| `events.orders.status_changed`       | logistics (auto-create shipment on `confirmed`), analytics, notifications |
| `events.orders.cancelled`            | inventory (release stock), analytics, notifications |
| `events.inventory.stock_changed`     | analytics                            |
| `events.inventory.low_stock`         | notifications, analytics             |
| `events.logistics.shipment_*`        | analytics, notifications             |
| `events.analytics.anomaly_detected`  | notifications                        |

---

## 9. Development Phases

### Constraints

- **Total time:** 14 calendar days (2026-04-11 → 2026-04-25)
- **Approach:** Vertical slices — each phase delivers end-to-end working functionality
- **Priority:** Working MCP orchestration is the diploma's core value

---

### Phase 1: Foundation (Days 1–3)

**Goal:** Infrastructure, auth, project skeleton, basic CRUD

| Task                                          | Details                                                       |
| --------------------------------------------- | ------------------------------------------------------------- |
| Docker Compose setup                          | PostgreSQL, Redis, NATS, all service containers                |
| API Gateway                                   | JWT validation, routing, CORS, rate limiting                   |
| User Service                                  | Registration (admin-only), login, JWT, RBAC middleware          |
| Database migrations framework                 | golang-migrate setup for all services                          |
| Project structure                             | Monorepo: `/services/{name}/`, `/frontend/`, `/mcp-orchestrator/`, `/gateway/`, `/docker/` |
| Frontend skeleton                             | Next.js project, layout, auth flow (login page, token storage)  |
| CI: Docker build for all services             | `docker-compose up` starts entire system                        |

**Deliverable:** User can log in; gateway routes requests; all services start and connect to their DBs.

---

### Phase 2: Core Business Services (Days 4–7)

**Goal:** All microservices with full CRUD + events

| Task                                          | Details                                                        |
| --------------------------------------------- | -------------------------------------------------------------- |
| Order Service                                 | Full CRUD, status transitions, events to NATS                   |
| Inventory Service                             | Products, stock, warehouses, movements, reservation, low stock   |
| Logistics Service                             | Shipments, carriers, mock routes, status tracking                |
| Notification Service                          | In-app notifications, WebSocket delivery, mock email/SMS adapters |
| Analytics Service                             | Event consumers, pre-aggregated tables, report endpoints          |
| NATS event wiring                             | All publishers and consumers connected                            |
| Seed data scripts                             | Realistic mock data for all services                              |
| Frontend: Core pages                          | Orders list/detail, Products, Stock, Shipments, Notifications     |

**Deliverable:** Full CRUD via UI; events flowing; seed data loaded; all services operational.

---

### Phase 3: MCP Orchestrator (Days 8–11)

**Goal:** AI layer fully functional

| Task                                          | Details                                                         |
| --------------------------------------------- | --------------------------------------------------------------- |
| MCP Gateway (WebSocket)                       | Auth, role extraction, message routing                           |
| Tool Registry                                 | All tools registered with schemas and role requirements           |
| Intent Parser                                 | LLM integration for intent classification and parameter extraction |
| Planner (DAG builder)                         | Multi-step workflow generation from parsed intent                  |
| Executor                                      | Sequential/parallel step execution via HTTP calls to Go services   |
| Context Manager                               | Redis-based conversation context for multi-turn chat               |
| Response Aggregator                           | LLM-formatted responses from tool execution results                |
| RBAC in MCP                                   | Role-based tool filtering and execution validation                  |
| Frontend: MCP Chat panel                      | WebSocket chat UI, message history, streaming responses             |

**Deliverable:** User can type natural language requests; MCP orchestrates multi-service workflows; role restrictions enforced.

---

### Phase 4: Dashboards, Analytics & Polish (Days 12–14)

**Goal:** Visual analytics, full integration, demo-ready

| Task                                          | Details                                                         |
| --------------------------------------------- | --------------------------------------------------------------- |
| Dashboard page                                | Role-specific KPI widgets with charts                            |
| Analytics page                                | Sales trends, inventory health, logistics performance charts      |
| Anomaly detection                             | Rule-based threshold detection, alerts in dashboard               |
| Supply optimization suggestions               | Basic algorithm, accessible via MCP and UI                        |
| Notification preferences                      | User settings for notification types                              |
| End-to-end testing                            | Full workflow: order → inventory → shipment → analytics → MCP      |
| Demo scenarios                                | 3–5 scripted MCP conversations for diploma defense                 |
| Bug fixes and polish                          | UI cleanup, error handling, loading states                         |
| Documentation                                 | README, API docs (Swagger/OpenAPI), architecture diagram            |

**Deliverable:** Demo-ready platform with all features working end-to-end.

---

## 10. Potential Challenges & Mitigations

| Challenge                                | Risk    | Mitigation                                                          |
| ---------------------------------------- | ------- | ------------------------------------------------------------------- |
| LLM response quality                     | High    | Detailed system prompts; typed tool schemas; fallback to "I don't understand" |
| 2-week deadline                          | High    | Vertical slices; cut analytics depth before cutting core features    |
| MCP DAG complexity                       | Medium  | Start with sequential execution; add parallel later                  |
| Inter-service data consistency           | Medium  | Idempotent event handlers; retry on NATS; eventual consistency is OK  |
| Gemini API rate limits / cost            | Medium  | Cache common intent patterns; mock LLM for development; Gemini Flash has generous free tier |
| Frontend scope                           | Medium  | Use component libraries (shadcn/ui); skip animations; function > form |
| Docker resource usage (6+ services)      | Low     | Minimal base images; shared PostgreSQL with separate schemas          |

---

## 11. Future Expansion (Post-MVP)

| Feature                          | Description                                                   |
| -------------------------------- | ------------------------------------------------------------- |
| Real external integrations       | Replace mock adapters with Google Maps, SendGrid, Twilio       |
| ClickHouse analytics             | Move heavy analytics to columnar DB                            |
| Kubernetes deployment            | Helm charts, horizontal scaling, health probes                  |
| Multi-tenancy                    | Organization-based data isolation                              |
| Advanced MCP capabilities        | Learning from user corrections; workflow templates; scheduling   |
| Audit log service                | Comprehensive audit trail across all services                   |
| Mobile app                       | React Native for logistics managers on the go                   |
| Real-time tracking               | WebSocket-based live shipment tracking on map                   |
| ML-based demand forecasting      | Replace rule-based analytics with trained models                |
| Webhook integrations             | Allow external systems to subscribe to events                   |

---

## 12. Infrastructure & Deployment

### 12.1 Docker Compose Services

```yaml
services:
  # Infrastructure
  postgres:       # PostgreSQL 16 (shared, multiple schemas)
  redis:          # Redis 7
  nats:           # NATS server

  # Application
  gateway:        # API Gateway (Go)
  user-service:   # User management (Go)
  order-service:  # Order management (Go)
  inventory-service: # Inventory management (Go)
  logistics-service: # Logistics (Go)
  analytics-service: # Analytics (Go)
  notification-service: # Notifications (Go)
  mcp-orchestrator: # MCP AI layer (Python)
  frontend:       # Next.js app

  # Observability (optional)
  prometheus:     # Metrics collection
  grafana:        # Dashboards
```

### 12.2 Port Allocation

| Service              | Internal Port | External (dev) |
| -------------------- | ------------- | -------------- |
| Gateway              | 8080          | 8080           |
| User Service         | 8001          | —              |
| Order Service        | 8002          | —              |
| Inventory Service    | 8003          | —              |
| Logistics Service    | 8004          | —              |
| Analytics Service    | 8005          | —              |
| Notification Service | 8006          | —              |
| MCP Orchestrator     | 8010          | —              |
| Frontend             | 3000          | 3000           |
| PostgreSQL           | 5432          | 5432           |
| Redis                | 6379          | 6379           |
| NATS                 | 4222          | 4222           |
| Prometheus           | 9090          | 9090           |
| Grafana              | 3001          | 3001           |

### 12.3 Shared PostgreSQL Strategy

Single PostgreSQL instance with separate schemas per service:

```sql
CREATE SCHEMA users;
CREATE SCHEMA orders;
CREATE SCHEMA inventory;
CREATE SCHEMA logistics;
CREATE SCHEMA analytics;
CREATE SCHEMA notifications;
```

Each service connects with its own credentials and has access only to its schema.

---

## 13. API Design Conventions

### 13.1 REST Endpoints Pattern

```
GET    /api/v1/{resource}          — List (with query params for filter/sort/page)
GET    /api/v1/{resource}/{id}     — Get by ID
POST   /api/v1/{resource}          — Create
PUT    /api/v1/{resource}/{id}     — Update
DELETE /api/v1/{resource}/{id}     — Soft delete
```

### 13.2 Standard Response Format

**Success:**

```json
{
  "data": { ... },
  "meta": {
    "total": 100,
    "limit": 20,
    "offset": 0
  }
}
```

**Error:**

```json
{
  "error": {
    "code": "order_not_found",
    "message": "Order with ID 'abc' not found",
    "data": { "order_id": "abc" }
  }
}
```

### 13.3 Pagination

Query parameters: `?limit=20&offset=0`

### 13.4 Filtering

Query parameters: `?status=pending&date_from=2026-01-01&date_to=2026-03-31`

### 13.5 Sorting

Query parameters: `?sort_by=created_at&sort_order=desc`

---

## 14. Demo Scenarios for Diploma Defense

### Scenario 1: End-to-End Order Flow (UI)

1. Admin creates a product → stock added to warehouse
2. Operator creates an order → stock reserved
3. Order confirmed → shipment auto-created
4. Logistics manager updates shipment status through lifecycle
5. Analytics dashboard reflects all changes in real-time

### Scenario 2: MCP Multi-Step Workflow

**User (Analyst):** "Show me sales summary for the last month, identify products with declining demand, and suggest inventory optimization"

**System:**
1. Calls `analytics.sales.summary` → gets revenue and top products
2. Calls `analytics.sales.trends` → identifies declining products
3. Calls `analytics.optimization.suggest` → generates reorder recommendations
4. Aggregates and presents formatted report

### Scenario 3: MCP Cross-Service Operation

**User (Admin):** "Find all orders that are late for delivery, notify the logistics manager, and create a report"

**System:**
1. Calls `logistics.shipments.list` (overdue filter)
2. Calls `users.get` (role=logistics_manager)
3. Calls `notifications.send` (to logistics manager)
4. Calls `analytics.report.generate` (late deliveries)
5. Returns summary with actions taken

### Scenario 4: RBAC Enforcement

**User (Operator):** "Delete all products from inventory"

**System:** "You don't have permission to manage inventory. As an Operator, you can work with orders. Would you like to see your current orders instead?"

### Scenario 5: Anomaly Detection

**Dashboard shows:** Spike in order cancellations detected
**User (Analyst) via MCP:** "Investigate the cancellation spike"
**System:** Analyzes patterns, identifies affected products/regions, suggests root causes

---

## 15. Glossary

| Term               | Definition                                                                              |
| ------------------ | --------------------------------------------------------------------------------------- |
| **MCP**            | Model Context Protocol — AI orchestration layer that translates natural language to service calls |
| **DAG**            | Directed Acyclic Graph — execution plan where steps can run sequentially or in parallel    |
| **Intent**         | Parsed user request containing action type, target service, and parameters                 |
| **Tool**           | Registered MCP capability that maps to a specific API endpoint on a microservice           |
| **Adapter**        | Interface abstraction for external integrations (mock in MVP, replaceable with real implementations) |
| **Eventual Consistency** | Data synchronization model where services become consistent over time via events      |
| **Soft Delete**    | Marking records as deleted (`deleted_at` timestamp) instead of physical removal             |
| **Seed Data**      | Pre-generated realistic mock data for development and demonstration                        |
