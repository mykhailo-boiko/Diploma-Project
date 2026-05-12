# Observability — Logfire, traces, audit

ChainOrchestra emits three layers of telemetry that let an operator (or AI
agent) reconstruct what happened for any given request:

1. **Logfire spans** — chat turns, Gemini calls, tool calls with attributes
   (`session_id`, `user_id`, `trace_id`, `order_id`, `shipment_id`, …).
2. **Audit log** — every write operation across services stored in
   `audit.action_log` with `actor_email`, `service_name`, `action`,
   `entity_ids[]`, `trace_id`, `result_status`.
3. **Prometheus metrics** — request rates, durations, per-actor simulator
   counters.

## Logfire setup

1. Create a free account at https://logfire.pydantic.dev/
2. Create a project and copy the write token
3. Add to `.env`:
   ```
   LOGFIRE_TOKEN=lf_xxx_xxxxxxxxxxxxx
   LOGFIRE_ENVIRONMENT=development
   ```
4. `docker compose up -d mcp-host` — spans will start streaming

If `LOGFIRE_TOKEN` is unset everything still works: spans become local no-ops
and Logfire calls are skipped. Suitable for diploma offline demos.

## Span hierarchy

```
chat.turn (session_id, user_id, trace_id, message_preview)
└── gemini.generate_content (model, input_tokens, output_tokens, finish_reason)
└── tool.call (tool_name, order_id?, shipment_id?, product_id?, error?)
```

## Useful Logfire queries

Last 50 LLM calls with errors:
```sql
SELECT trace_id, attributes->>'tool_name' AS tool, attributes->>'error_code' AS code
FROM records
WHERE name = 'tool.call' AND attributes->>'error' = 'true'
ORDER BY start_time DESC LIMIT 50
```

All calls touching a specific order_id:
```sql
SELECT name, attributes, duration_ns / 1e6 AS ms
FROM records
WHERE attributes->>'order_id' = '550e8400-e29b-41d4-a716-446655440000'
ORDER BY start_time
```

Sessions sorted by total tokens (cost watch):
```sql
SELECT
  attributes->>'session_id' AS session,
  SUM((attributes->>'total_tokens')::int) AS tokens
FROM records
WHERE name = 'gemini.generate_content'
GROUP BY session ORDER BY tokens DESC LIMIT 20
```

p95 latency per tool:
```sql
SELECT
  attributes->>'tool_name' AS tool,
  percentile_cont(0.95) WITHIN GROUP (ORDER BY duration_ns / 1e6) AS p95_ms
FROM records
WHERE name = 'tool.call'
GROUP BY tool ORDER BY p95_ms DESC
```

## Trace IDs

Every chat turn generates a UUID v4 `trace_id` on the frontend
(`use-chat-socket.ts`) and ships it as part of the WebSocket message. The
MCP host attaches it as a span attribute. Frontend can fetch the timeline
of all `audit.action_log` events for a given order/shipment via the
**Trace viewer** page: `/admin/trace/[entityId]`.

Backend services receive `X-Trace-ID` from the gateway proxy middleware and
write it to `audit.action_log.trace_id`. The trace_id therefore links:
**Logfire spans ↔ audit log rows ↔ user-visible UI events**.

## Token budgets

- **Per-session cap** (default 200_000 tokens): `MAX_TOKENS_PER_SESSION` env
- **Per-user-per-hour cap** (default 1_000_000): `MAX_TOKENS_PER_USER_PER_HOUR`

When exceeded, the next LLM round inside the same WebSocket is aborted with a
clear message to the user. Counters stored in Redis (`budget:session:{id}`,
`budget:user:{id}:hour`).

GET `/api/v1/mcp/budget/{session_id}` returns current usage / cap / exceeded flag.

## Loop detection

The `LoopGuard` watches the last 8 tool invocations per session. If the same
`(tool_name, hash(sorted_args))` appears ≥3 times the next call is replaced
with a structured `loop_detected` error containing a `suggestion` that
directs the LLM to change approach instead of retrying.
