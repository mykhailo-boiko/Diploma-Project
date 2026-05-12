# MCP Tools — Strict Pydantic Typing

All 93 MCP tools use Pydantic-typed parameters via `typing.Annotated` so
FastMCP can emit precise JSON Schemas that Gemini honours when generating
tool calls. The validators carry friendly, actionable error messages that
the LLM can read and self-correct on the next turn.

The shared types live in [`mcp-orchestrator/types_mcp.py`](../mcp-orchestrator/types_mcp.py).

## Core types

| Type | Description | Example value |
|------|-------------|---------------|
| `UUIDStr` | UUID in 8-4-4-4-12 hex format | `550e8400-e29b-41d4-a716-446655440000` |
| `TrackingNumber` | Public tracking code | `CO-2026-K7H2P9` |
| `ISODate` | ISO 8601 date | `2026-05-12` |
| `ISODateTime` | RFC3339 timestamp | `2026-05-12T15:30:00Z` |
| `PhoneE164` | E.164 phone | `+380501112233` |
| `EmailAddr` | Email | `user@example.com` |
| `PositiveInt` | int > 0 | `5` |
| `NonNegativeInt` | int ≥ 0 | `0` |
| `PageLimit` | 1..1000 | `100` |
| `PageOffset` | ≥ 0 | `0` |
| `Money` | 0..10M float | `19.99` |

## Enums (Literal types)

`OrderStatus`, `ShipmentStatus`, `CarrierType`, `StockMovementType`,
`NotificationType`, `NotificationPriority`, `NotificationChannel`,
`UserRole`, `SortOrder`, `SimulatorScenario`, `AnalyticsMetric`,
`ForecastMethod`, `AnomalyCategory`, `ProductCategory`.

When the LLM passes an invalid enum value, the response includes the full
allowed list, so it can pick the correct one on retry.

## What this fixes

| Before | After |
|--------|-------|
| `date_from: str` — Gemini may send `"May 2026"` and backend returns generic "validation error" | `date_from: ISODate` — Gemini sees `format: date` schema, sends `2026-05-01`; if wrong, error explains what's expected |
| `status: str` — any string accepted by schema | `status: OrderStatus` — only the 8 enum values, schema shows `enum: [pending, confirmed, ...]` |
| `order_id: str` — any string | `order_id: UUIDStr` — pattern-validated; tracking number passed by mistake gets a friendly error |
| `phone: str = None` | `phone: PhoneE164` — must start with `+`, 7-15 digits |

## How a validation failure flows back to the LLM

1. LLM generates `analytics_sales(date_from="2026/05/12", date_to="2026-05-15")`
2. Pydantic raises `ValidationError` with message
   `must be ISO 8601 date in format 'YYYY-MM-DD' (e.g. '2026-05-12'); received: '2026/05/12'`
3. FastMCP serializes the error into the tool result envelope
4. LLM reads it on the next round and retries with `date_from="2026-05-12"`

Unit tests in [`tests/test_typed_validators.py`](../mcp-orchestrator/tests/test_typed_validators.py)
cover all primitives with valid / invalid / edge cases.
