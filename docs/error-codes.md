# LLM-friendly Error Codes

All ChainOrchestra services return errors in a uniform envelope that LLM agents
can parse for self-correction:

```json
{
  "error": {
    "code": "validation_error",
    "message": "field 'date_from' must be ISO date YYYY-MM-DD",
    "data": {
      "field": "date_from",
      "expected": "string in format YYYY-MM-DD",
      "received": "2026/05/12",
      "suggestion": "Use the format 2026-05-12 (year-month-day with hyphens)",
      "examples": ["2026-05-01", "2026-12-31"]
    }
  }
}
```

The MCP orchestrator catches HTTP errors and wraps them in this shape before
returning to the LLM as the tool result (no exceptions are raised — LLM always
gets a structured `{"error": {...}}` dict that it can read and retry).

## Code reference

| Code | HTTP | Meaning | LLM action |
|------|------|---------|------------|
| `validation_error` | 400 | Generic validation failure | Read `expected`/`received`, fix and retry |
| `missing_field` | 400 | Required field absent | Add the field per `suggestion` |
| `invalid_field` | 400 | Field present but wrong type/value | Read `expected`+`examples`, retry |
| `invalid_body` | 400 | Body could not be JSON-parsed | Check JSON syntax |
| `invalid_status_transition` | 409 | State machine refused transition | Fetch entity, choose valid next status |
| `cannot_cancel` | 409 | Entity already in terminal state | Inform user |
| `auth_required` | 401 | Token expired or missing | Report — internal issue |
| `forbidden` | 403 | Role-insufficient | Inform user that admin role needed |
| `not_found` | 404 | Entity does not exist | Verify ID via `*_list`/`*_get` |
| `conflict` | 409 | Operation conflicts with current state | Fetch state, adjust request |
| `unprocessable_entity` | 422 | Semantic validation failed | Read error details, fix values |
| `rate_limit_exceeded` | 429 | Too many requests | Slow down |
| `internal_error` | 500 | Backend bug | Do not retry; report to user |
| `bad_gateway` | 502 | Upstream unreachable | Report; do not retry within turn |
| `service_unavailable` | 503 | Backend starting/maintenance | Do not retry within turn |
| `request_timeout` | 504 | Backend slow | Narrow filter / smaller page |
| `loop_detected` | – | MCP host caught a tool-loop | Stop calling same tool with same args |
| `budget_exceeded` | – | Token budget for session exceeded | Tell user to start a new chat |
| `transport_error` | – | Network failure to backend | Do not retry blindly |

`code` values may also come unchanged from the backend (e.g. `carrier_not_found`,
`stock_not_found`, `insufficient_stock`) — every such code has `data.suggestion`.

## Example flows

### Wrong date format
1. LLM calls `analytics_sales(date_from="2026/05/12", date_to="2026-05-15")`
2. MCP returns `{error: {code:"validation_error", field:"date_from",
   expected:"string in format YYYY-MM-DD", received:"2026/05/12",
   suggestion:"Use 2026-05-12...", examples:["2026-05-01"]}}`
3. LLM reads `suggestion`, retries with `date_from="2026-05-12"` → success.

### Invalid status transition
1. LLM calls `orders_update_status(order_id, status="delivered")` on a
   `pending` order
2. MCP returns `{error: {code:"invalid_status_transition", field:"status",
   suggestion:"From 'pending', allowed next: 'confirmed', 'cancelled'",
   examples:["confirmed","cancelled"]}}`
3. LLM tells the user the order needs to be confirmed first.

### Tool loop
1. LLM keeps calling `orders_search("foo")` 3+ times with identical args
2. LoopGuard injects `{error: {code:"loop_detected", message:"called
   orders_search 3 times with same arguments", suggestion:"Stop calling
   this tool with these arguments. Try different tool / args / summarize"}}`
3. LLM stops, summarizes for the user.
