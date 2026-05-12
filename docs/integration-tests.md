# Integration Tests

Live end-to-end tests against the running Docker Compose stack.
Located in [`tests/integration/`](../tests/integration/).

## Running

```bash
docker compose up -d
ADMIN_PASSWORD='upB@7UmdvEWYe&t#8nY%' ./scripts/run-integration-tests.sh -v
```

To opt in to the LLM E2E test (uses Gemini quota):
```bash
E2E_LLM=1 ADMIN_PASSWORD='...' ./scripts/run-integration-tests.sh -v -k chat
```

## Test files

| File | Scenarios |
|------|-----------|
| `test_order_full_lifecycle.py` | pending→confirmed→processing→shipped→delivered, audit trail check, invalid transition, cancel |
| `test_shipment_postal_pipeline.py` | 15-state postal pipeline, manual events, recipient delivery, 3 attempts → returns to sender |
| `test_typed_validation.py` | bad date format, missing field, invalid stock type, negative quantity all return structured errors |
| `test_sse_realtime.py` | SSE stream emits business events when simulator runs |
| `test_simulator_admin.py` | start/stop/scenario/speed admin endpoints + 403 for non-admin |
| `test_audit_trace.py` | trace by entity_id, audit_log query, missing entity_id returns validation_error |
| `test_correlation_trace.py` | `X-Trace-ID` propagates to `audit.action_log.trace_id` |
| `test_rate_limit_and_loop.py` | gateway tolerates burst, budget endpoint returns status |
| `test_llm_chat_flow.py` | (E2E_LLM=1) WS chat → tool call → final message |

## Conventions

- Tests acquire admin token via `ADMIN_EMAIL` / `ADMIN_PASSWORD` env (admin
  fixture in `conftest.py`).
- They use `httpx` for HTTP, `websockets` for the chat WS test.
- Test fixtures (`pick_first_product`, `pick_first_warehouse`,
  `pick_first_active_carrier`) skip gracefully if data is missing.
- Each test cleans up only what it must; the stack is designed to survive
  arbitrary repetition.
