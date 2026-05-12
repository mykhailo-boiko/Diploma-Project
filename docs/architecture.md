# Architecture

## System Overview

ChainOrchestra is a microservice-based supply chain management platform with an AI orchestration layer. The system consists of 15 containers orchestrated via Docker Compose, including a dedicated simulator-service that generates continuous traffic to validate the platform under real-world load.

```mermaid
graph TB
    subgraph "Client Layer"
        FE["Frontend<br/>(Next.js 16)<br/>:3000"]
    end

    subgraph "Gateway Layer"
        GW["API Gateway<br/>(Go)<br/>:8080"]
        MH["MCP Host<br/>(FastAPI/WebSocket)<br/>:8090"]
    end

    subgraph "AI Layer"
        MO["MCP Orchestrator<br/>(FastMCP)<br/>93 tools"]
        LLM["Google Gemini<br/>LLM"]
    end

    subgraph "Business Services"
        US["User Service<br/>:8001"]
        OS["Order Service<br/>:8002"]
        IS["Inventory Service<br/>:8003"]
        LS["Logistics Service<br/>:8004"]
        AS["Analytics Service<br/>:8005"]
        NS["Notification Service<br/>:8006"]
        SIM["Simulator Service<br/>:8007"]
    end

    subgraph "Infrastructure"
        PG["PostgreSQL 16<br/>:5432<br/>6 schemas"]
        RD["Redis 7<br/>:6379"]
        NT["NATS 2<br/>:4222"]
    end

    FE -->|REST API| GW
    FE -->|WebSocket| MH

    GW -->|Proxy + JWT headers| US
    GW -->|Proxy + JWT headers| OS
    GW -->|Proxy + JWT headers| IS
    GW -->|Proxy + JWT headers| LS
    GW -->|Proxy + JWT headers| AS
    GW -->|Proxy + JWT headers| NS
    GW -->|Proxy + admin-only| SIM
    SIM -->|REST as admin| GW

    MH -->|stdio| MO
    MH -->|API calls| LLM
    MH -->|Session store| RD
    MO -->|HTTP via gateway| GW

    US --> PG
    OS --> PG
    IS --> PG
    LS --> PG
    AS --> PG
    NS --> PG

    OS -->|Publish events| NT
    IS -->|Publish events| NT
    LS -->|Publish events| NT

    NT -->|Subscribe| AS
    NT -->|Subscribe| NS
    NT -->|Subscribe| LS
    NT -->|Subscribe| IS
```

## Request Flow

### Traditional UI Flow

```mermaid
sequenceDiagram
    participant U as User (Browser)
    participant F as Frontend
    participant G as API Gateway
    participant S as Go Service
    participant D as PostgreSQL

    U->>F: Click action
    F->>G: HTTP request + JWT
    G->>G: Validate JWT
    G->>G: Rate limit check
    G->>S: Proxy + X-User-ID, X-User-Role headers
    S->>S: Middleware extracts user context
    S->>D: Query/Mutation
    D-->>S: Result
    S-->>G: JSON response
    G-->>F: Proxied response
    F-->>U: Updated UI
```

### MCP Chat Flow

```mermaid
sequenceDiagram
    participant U as User
    participant F as Frontend
    participant H as MCP Host
    participant L as Gemini LLM
    participant M as MCP Orchestrator
    participant G as API Gateway
    participant S as Go Service

    U->>F: "Create an order for John"
    F->>H: WebSocket message + JWT
    H->>H: Validate JWT, determine role
    H->>H: Filter tools by RBAC
    H->>L: User message + tool declarations
    L-->>H: function_call: orders_create(...)
    H->>M: Execute tool via MCP protocol
    M->>G: POST /api/v1/orders
    G->>S: Proxy to order-service
    S-->>G: Order created
    G-->>M: Response
    M-->>H: Tool result
    H->>L: Tool result
    L-->>H: "I created order #abc for John..."
    H-->>F: WebSocket response (streaming)
    F-->>U: Chat message displayed
```

## Database Schema Layout

Each service owns its own PostgreSQL schema within the shared `chainorchestra` database:

| Schema | Service | Tables |
|--------|---------|--------|
| `users` | user-service | `users`, `password_reset_tokens` |
| `orders` | order-service | `orders`, `order_items` |
| `inventory` | inventory-service | `products`, `warehouses`, `stock`, `stock_movements` |
| `logistics` | logistics-service | `carriers`, `shipments`, `routes` |
| `analytics` | analytics-service | `sales_daily`, `inventory_snapshot`, `logistics_daily` |
| `notifications` | notification-service | `notifications`, `notification_preferences` |

## NATS Event Flow

```mermaid
graph LR
    subgraph "Publishers"
        OS["Order Service"]
        IS["Inventory Service"]
        LS["Logistics Service"]
    end

    subgraph "NATS Subjects"
        OC["order.created"]
        OSC["order.status_changed"]
        OCA["order.cancelled"]
        ISC["inventory.stock_changed"]
        ILS["inventory.low_stock"]
        LSC["logistics.shipment_created"]
        LSS["logistics.shipment_status_changed"]
    end

    subgraph "Subscribers"
        AS["Analytics Service"]
        NS["Notification Service"]
        LS2["Logistics Service<br/>(consumer)"]
        IS2["Inventory Service<br/>(consumer)"]
    end

    OS --> OC
    OS --> OSC
    OS --> OCA
    IS --> ISC
    IS --> ILS
    LS --> LSC
    LS --> LSS

    OC --> AS
    OC --> NS
    OSC --> AS
    OSC --> NS
    OSC --> LS2
    OCA --> AS
    OCA --> NS
    OCA --> IS2
    ISC --> AS
    ILS --> AS
    ILS --> NS
    LSC --> AS
    LSC --> NS
    LSS --> AS
    LSS --> NS
```

## RBAC Model

```mermaid
graph TD
    subgraph "Roles"
        A["Admin"]
        O["Operator"]
        WM["Warehouse Manager"]
        LM["Logistics Manager"]
        AN["Analyst"]
    end

    subgraph "Service Access"
        ALL["All Services"]
        ORD["Orders"]
        INV["Inventory"]
        LOG["Logistics"]
        ANA["Analytics"]
        NOT["Notifications"]
        USR["Users (admin)"]
    end

    A --> ALL
    O --> ORD
    O --> NOT
    WM --> INV
    WM --> ORD
    WM --> NOT
    LM --> LOG
    LM --> ORD
    LM --> NOT
    AN --> ANA
    AN --> NOT
```

## Middleware Chain

Every request through the API Gateway passes through:

```
RequestID → CORS → Recovery → Logging → RateLimit → JWT → Proxy → Service
```

Each Go service then applies:

```
UserContext (extract X-User-ID/X-User-Role from headers) → Handler
```

## Technology Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Go for services | Go 1.25 + net/http | Performance, strong typing, simple deployment |
| PostgreSQL schemas | Single DB, multiple schemas | Isolation without operational overhead |
| NATS | NATS 2 with JetStream | Lightweight, Go-native, persistent messaging |
| MCP protocol | FastMCP (Python) | Standard protocol for LLM tool integration |
| Gemini | google-genai SDK | Function calling support, fast inference |
| Next.js | v16 with App Router | Modern React patterns, SSR-ready |
| Redis | v7 | Session storage, plan caching, TTL support |
