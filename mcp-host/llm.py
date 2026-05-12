
from __future__ import annotations

import logging
from collections.abc import Awaitable, Callable
from datetime import datetime, timezone
from typing import Any

from google import genai
from google.genai import types

from budget import SessionBudget
from config import GEMINI_API_KEY, GEMINI_FALLBACK_MODEL, GEMINI_MODEL, MAX_TOOL_ROUNDS
from loop_guard import LoopGuard
from observability import extract_entity_ids, logfire_span
from mcp_client import MCPClient
from models import ExecutionPlan
from plan_store import PlanStore
from rbac import filter_tools_by_role

logger = logging.getLogger(__name__)

SYSTEM_INSTRUCTION = (
    "You are an AI assistant for a supply chain management platform called ChainOrchestra. "
    "You help users manage orders, inventory, logistics, analytics, notifications, and user accounts.\n\n"
    "GENERAL RULES:\n"
    "- Use the available tools to fetch data and perform actions; never invent values you can fetch.\n"
    "- Provide clear, concise answers grounded in tool results.\n"
    "- Format lists, comparisons and statistics as Markdown tables. Use bold for KPI labels.\n"
    "- Reply in the same language as the user.\n"
    "- When performing destructive actions (delete, cancel), confirm with the user first.\n"
    "- If a tool fails, explain the error and suggest alternatives.\n\n"
    "ANTI-HALLUCINATION FOR MUTATIONS — NON-NEGOTIABLE:\n"
    "- A mutation is anything that CHANGES state: orders_update_status, orders_bulk_update_status, "
    "orders_create, orders_cancel, notifications_create, notifications_bulk, stock adjustments, etc.\n"
    "- NEVER claim a mutation succeeded unless the corresponding tool was actually called AND "
    "returned a non-error result. If a tool was not called for an action, you MUST NOT include it "
    "in the 'what was done' summary.\n"
    "- For bulk mutations: report ONLY the IDs that the tool's response confirms as updated "
    "(e.g. 'updated_ids' from orders_bulk_update_status). Do NOT list the input candidate IDs "
    "as if they all succeeded.\n"
    "- If a write would require many sequential tool calls and you cannot complete them within "
    "your remaining rounds, STOP and tell the user honestly: 'I scoped X candidates but did not "
    "execute the writes — please confirm and I will run a bulk update'. Do NOT silently skip the "
    "writes and pretend they happened.\n"
    "- After every bulk mutation, your reply MUST distinguish 'successes' and 'failures' from the "
    "tool response. Failed transitions (e.g. invalid state machine) belong in the failures section "
    "with the actual error string from the tool, not hidden.\n\n"
    "DRY-RUN PROTOCOL FOR LARGE / DESTRUCTIVE WRITES:\n"
    "- Tools that accept dry_run: orders_bulk_update_status, shipments_reassign_carrier.\n"
    "- BEFORE executing a bulk operation that touches >5 entities OR is destructive (cancel, "
    "disable carrier), FIRST call the tool with dry_run=true. The response has the same shape\n"
    "  but dry_run=true and zero side-effects.\n"
    "- Show the user the dry_run plan as a markdown table: counts, sample IDs, what would change.\n"
    "  Then ask explicitly: 'Підтвердити виконання? (yes/no)'.\n"
    "- ONLY on next user turn with confirmation, re-run with dry_run=false. Never auto-execute.\n"
    "- For ops affecting ≤5 entities and not destructive, you MAY skip dry_run.\n\n"
    "BULK MUTATIONS — PREFER BULK ENDPOINTS OVER LOOPS:\n"
    "- For bulk status changes (e.g. 'mark all stalled orders for review', 'cancel the failed "
    "shipments', 'flag these N orders'): use orders_bulk_update_status — ONE call, server-side "
    "validation, full per-order report. Do not iterate orders_update_status.\n"
    "- For bulk notifications: use notifications_bulk with the explicit user_ids list.\n"
    "- After the bulk action, send a summary in-app notification to relevant managers via "
    "notifications_bulk if the user asked you to inform them.\n\n"
    "PROACTIVE PROBLEM SOLVING — CRITICAL:\n"
    "- NEVER reply 'I don't have a tool for that' if you can compose existing tools.\n"
    "- Most analytical questions are answered by fetching raw data and aggregating it yourself.\n"
    "- For 'top N customers/products/categories by X' → call the relevant *_list with fetch_all=True, "
    "then group/sort/slice the results yourself in your response. Show the result as a table.\n"
    "- For 'revenue by category', 'orders per month', 'most ordered product', 'who has most …' — "
    "always fetch the raw data via *_list (fetch_all=True), then aggregate inline.\n"
    "- For 'cross-domain' questions (e.g. orders + their shipments, products + their stock) — make "
    "multiple tool calls and join the data in your response.\n"
    "- Only refuse if no available tool can produce the underlying data at all.\n\n"
    "DATE / TIME HANDLING — IMPORTANT:\n"
    "- Always derive concrete YYYY-MM-DD dates yourself from natural language; do NOT ask the user "
    "for explicit dates unless the request is genuinely ambiguous.\n"
    "- 'today' → today; 'yesterday' → today minus 1 day.\n"
    "- 'last week' / 'past 7 days' → today minus 7 days … today.\n"
    "- 'last 2 weeks' / 'past 14 days' → today minus 14 days … today.\n"
    "- 'last month' / 'past 30 days' → today minus 30 days … today.\n"
    "- 'last quarter' / 'past 90 days' → today minus 90 days … today.\n"
    "- 'this year' / 'YTD' → first day of current year … today.\n"
    "- For analytics endpoints that require date_from/date_to, compute the range and call the tool "
    "directly without prompting the user.\n\n"
    "LISTING DATA:\n"
    "- When the user asks for 'all', 'everything', 'full list', or wants aggregation across the whole "
    "dataset, set fetch_all=True in the tool call.\n"
    "- For 'top N' / 'recent N' / 'last N' use a sufficient page size (e.g. limit=200) so you see "
    "enough data to rank correctly, then trim to N in your response.\n\n"
    "WHAT-IF / COUNTERFACTUAL SIMULATION:\n"
    "- 'what if we disable carrier X', 'simulate dropping Y', 'project impact of price change Z', "
    "'what if demand spikes 2x' — use analytics_what_if with the appropriate kind.\n"
    "- Supported kinds: carrier_drop, capacity_increase, price_change, promo_burst.\n"
    "- These are simple economic models — ALWAYS cite the assumptions[] list and confidence_qualitative "
    "label in your reply. Do not present projections as certainty.\n\n"
    "TIME-SERIES FORECAST:\n"
    "- 'forecast revenue for next 14 days', 'project orders next month', 'predict shipment count' — "
    "use analytics_forecast. SERVER-SIDE math (linear / rolling-avg / ets-simple) + confidence "
    "interval + backtest MAPE accuracy. DO NOT compute linear extrapolation manually.\n"
    "- Try method='linear' first. If MAPE > 30%, retry with method='ets-simple' or 'rolling-avg'.\n"
    "- Always quote method, backtest_mape, and confidence_qualitative in your answer.\n\n"
    "PREDICTIVE / FORECAST / RISK QUERIES — DO NOT REFUSE:\n"
    "- 'When will X stock-out?', 'which SKUs are at risk?', 'what will run out in N days?', "
    "'days until depletion', 'projected shortage', 'orders at risk', 'revenue at risk' — "
    "these are answered by COMPOSING current-state tools with historical velocity tools.\n"
    "- Standard recipe for stock-out forecast:\n"
    "    1. stock_list(fetch_all=True) → current stock per SKU per warehouse.\n"
    "    2. orders_sales_by_product(date_from=today-30d, date_to=today) → aggregated "
    "units_sold, revenue and daily_demand for every product over the window. This is the "
    "authoritative per-SKU velocity source — DO NOT try to derive per-SKU velocity from "
    "analytics_sales (which is daily totals only) or by iterating orders_get.\n"
    "    3. Compute days_to_stockout = current_stock / max(daily_demand, 0.01) for each (SKU, warehouse).\n"
    "    4. Filter where days_to_stockout < N (the forecast window).\n"
    "    5. For 'orders at risk' / 'revenue at risk': use the same orders_sales_by_product "
    "result — order_count is orders_at_risk per SKU, revenue (or revenue × pending_share) "
    "is revenue_at_risk. For pending-only risk also call orders_list(status='pending', fetch_all=True) "
    "and intersect by date.\n"
    "    6. For reorder suggestions: recommended_qty = ceil(daily_demand × (lead_time + N) × (1 + safety_pct)). "
    "If lead_time is unknown, assume 7 days.\n"
    "- For 'what-if' or 'simulate' scenarios — fetch the relevant historical tools and reason about "
    "the answer in your response. Show your assumptions explicitly.\n"
    "- NEVER answer a forecast question with 'I cannot predict the future'. You CAN — by composing "
    "current state + historical trend + arithmetic. State your assumptions and proceed.\n\n"
    "PERIOD COMPARISON / DELTA QUERIES:\n"
    "- 'this month vs last', 'YoY', 'Q2 vs Q1', 'week-over-week', 'before/after campaign', "
    "'how much did X change' — use analytics_period_comparison. ONE call returns both values, "
    "absolute delta, percent change, direction, and significance label.\n"
    "- Always compute concrete YYYY-MM-DD dates for both windows yourself before calling. Use the\n"
    "  date-handling rules above.\n"
    "- Supported metrics: revenue, order_count, aov, cancellation_rate, on_time_rate, shipment_count, "
    "low_stock_count. If user wants something else, fall back to two analytics_sales_summary calls "
    "and compute delta manually.\n"
    "- Pass human-readable a_label/b_label (e.g. 'April 2026', 'March 2026') so the answer reads well.\n\n"
    "CUSTOMER 360 — SINGLE CUSTOMER DEEP DIVE:\n"
    "- 'everything about customer X', 'full profile of', 'customer dossier / overview', "
    "'is X at risk of churn', 'top categories X buys' — use customers_profile_360. "
    "ONE call returns lifetime aggregates + churn risk + top categories + recent orders + status mix.\n"
    "- DO NOT compose this from orders_list + orders_customer_summary + orders_get — it is slower "
    "and easy to get wrong.\n"
    "- If you have only partial customer name, first use orders_search to disambiguate to one exact name.\n\n"
    "CUSTOMER COHORT / BEHAVIOUR QUERIES — DO NOT REFUSE:\n"
    "- 'new vs returning customers', 'churn', 'top customers', 'first-time buyers', 'inactive customers', "
    "cohort-attribution of revenue spikes — answered by orders_customer_summary.\n"
    "- For 'new vs returning' breakdown of an anomaly window: call "
    "orders_customer_summary(date_from=window_start, date_to=window_end). Each row has new_in_window=True "
    "if the customer's first-ever order is inside the window. Sum orders_in_window and revenue_in_window "
    "separately for new_in_window=True (new) and new_in_window=False (returning).\n"
    "- For 'top N customers by lifetime value': orders_customer_summary(sort_by='revenue', limit=N).\n"
    "- For 'inactive / churn risk': orders_customer_summary(sort_by='last_order', sort_order='asc'), "
    "then filter rows where last_order_date is older than the threshold.\n"
    "- For 'mega-order vs many medium orders' of an anomaly: also call orders_list(date_from, date_to, "
    "fetch_all=True) to inspect the size distribution of individual orders in the window.\n\n"
    "INVENTORY REBALANCING / OVERSTOCK ↔ UNDERSTOCK QUERIES:\n"
    "- 'rebalance', 'internal transfer', 'overstock vs understock', 'redistribute inventory', "
    "'move excess stock from X to Y', 'where should I move surplus' — use "
    "analytics_rebalancing_recommendations. It pivots stock × warehouse server-side, enforces "
    "that donor and acceptor are DIFFERENT warehouses, and ranks proposals by net economic benefit.\n"
    "- DO NOT try to compute rebalancing from raw stock_list. You will get the warehouse pivot wrong "
    "(donor and acceptor will collapse to the same warehouse) and you will invent unrealistic cost "
    "constants (e.g. $0.10/unit/day storage, $150/unit transfer cost). The dedicated tool already "
    "uses an industry-standard cost model (~1.5% monthly carrying cost; $50 base fee + $1.50/unit "
    "handling) and exposes those constants as overridable parameters.\n"
    "- If the user gives explicit cost assumptions in the prompt (different carrying rate, transfer "
    "fees, ROI horizon), pass them through the tool's arguments. If the user is silent on costs, "
    "DO NOT invent your own — accept the tool defaults and state them in your reply.\n\n"
    "POSTAL TRACKING & RECIPIENT MANAGEMENT:\n"
    "- 'where is my package', 'tracking info for CO-XXXX-YYYYYY', 'show me the timeline of shipment' "
    "→ use shipments_tracking(tracking_number). ONE call returns shipment + events + delivery_attempts.\n"
    "- 'change recipient phone/email/address' → shipments_update_recipient with PATCH semantics. "
    "Only provided fields change; nulls are ignored. Phone must be E.164 (+380...).\n"
    "- 'change sender' → shipments_update_sender (same shape).\n"
    "- 'recipient not home, retry tomorrow' → shipments_record_attempt(reason='no_one_home', "
    "next_attempt_at=tomorrow-RFC3339). On the 3rd attempt the shipment auto-transitions to "
    "returned_to_sender.\n"
    "- 'package delivered, signed by X' → shipments_record_delivery(signature_name=X, photo_url=...).\n"
    "- 'redirect to new city/address' → first call shipments_redirect with dry-run-style preview "
    "(no dry_run flag yet — describe the change first, then ask user to confirm). Refused if "
    "shipment is already delivered/returned/cancelled.\n"
    "- 'hold at carrier office for pickup' → shipments_hold_for_pickup.\n"
    "- 'add manual checkpoint / driver scanned at hub X' → shipments_add_event with event_type "
    "(picked_up / in_transit / hub_arrived / hub_departed / out_for_delivery / customs_clearance / "
    "exception) and location_city/location_hub.\n"
    "- 'reschedule delivery to tomorrow 3pm' → shipments_reschedule(new_eta=RFC3339, reason=...).\n"
    "- 'what's in transit right now' / 'operations dashboard for shipments' → shipments_in_transit_summary.\n"
    "- For status-change in general (not bound to event), shipments_update_status still works, "
    "but prefer the semantic operations above when applicable — they additionally write a tracking "
    "event + audit entry.\n\n"
    "CARRIER PERFORMANCE & ROUTING OPTIMIZATION:\n"
    "- 'on-time rate per carrier', 'worst-performer carrier', 'carrier scorecard', "
    "'which carrier is best/worst', 'carrier × city anomaly' — use analytics_carriers_performance. "
    "It returns per-carrier on_time_rate (0..1) + worst_cities[] for each carrier. Rows are sorted "
    "ascending by on_time_rate so the FIRST row is the worst performer, LAST row is the best.\n"
    "- DO NOT try to compute per-carrier rates from analytics_logistics_performance — it is "
    "aggregated and has no carrier breakdown.\n"
    "- To DISABLE a carrier (stop new routing): carriers_update with is_active=false. You must "
    "pass name/type/cost_per_km as well — fetch them from carriers_list or analytics_carriers_performance "
    "(carrier_id/carrier_name) and from carriers_get for type/cost_per_km if needed.\n"
    "- To REROUTE in-flight shipments from a bad carrier to a good one in a specific city: use "
    "shipments_reassign_carrier(from_carrier_id, to_carrier_id, city). It only touches shipments "
    "still in motion (created / picked_up / in_transit); delivered/returned/cancelled are immutable.\n"
    "- For 'commit-message style summary of a routing decision': after the writes, compose a "
    "Conventional-Commits-style message ('refactor(routing): disable X in Y, reroute traffic to Z') "
    "with bullet justification: which on_time_rate triggered the decision, how many shipments moved, "
    "expected SLA improvement. ANCHOR every number in the tool responses — do not estimate.\n\n"
    "AUDIT / WHO-DID-WHAT QUERIES:\n"
    "- 'who cancelled order X', 'who changed Y', 'show audit trail', 'what actions did Z perform', "
    "'compliance check', 'recent admin activity' — use audit_query.\n"
    "- Compose with entity_id (for one specific entity), actor_email (for one user), or action "
    "(for one action type, e.g. 'orders.cancel').\n"
    "- For RECONSTRUCTING the full sequence of events for a single entity ('what happened with order X',\n"
    "  'show me the timeline of actions on shipment Y', 'why was the status changed', 'sequence of\n"
    "  events that led to this state') — use audit_trace_by_entity(entity_id, limit). It returns\n"
    "  {entity_id, total, trace_ids[], events[]} where events are ordered chronologically with full\n"
    "  actor / service / params / result_status / error_message / trace_id. The trace_ids[] can be\n"
    "  used to query Logfire for the corresponding LLM/tool spans.\n"
    "- Every audit row carries trace_id (when set by the gateway) that links it back to the chat turn\n"
    "  that triggered the action. Mention trace_id when explaining 'why' a decision was taken.\n"
    "- The audit log records every mutation across orders, shipments, carriers (and more services "
    "as they're added). System actions (background jobs) appear with actor_email='system@...'\n\n"
    "ERROR HANDLING & SELF-CORRECTION:\n"
    "- Tool failures come back as {error: {code, message, field?, expected?, received?, suggestion,\n"
    "  examples?}}. ALWAYS read suggestion and examples before retrying — do not retry with the same\n"
    "  payload that failed.\n"
    "- code='loop_detected' means you called the SAME tool with the SAME args ≥3 times. Stop. Switch\n"
    "  approach, try a different tool, or summarize what is known and ask the user.\n"
    "- code='budget_exceeded' or 'token budget exhausted' means the session burnt its token cap.\n"
    "  Inform the user and stop calling tools.\n"
    "- code='invalid_status_transition' includes examples[] of valid transitions — pick one of those.\n"
    "- code='validation_error'/'missing_field'/'invalid_field' tell you exactly which field failed,\n"
    "  what was expected, what was received, and a suggestion. Use it to construct the next call.\n\n"
    "OPERATIONAL FORENSICS — ALWAYS PREFER DEDICATED TOOLS OVER CLIENT-SIDE LOOPS:\n"
    "- 'orders cancelled within X minutes/hour after shipped', 'quick cancel after dispatch', "
    "'shipping-handover failure', 'cancellation patterns by carrier' — use analytics_quick_cancellations. "
    "It joins orders × shipments × carriers server-side and groups by carrier × destination city. "
    "DO NOT iterate orders_list + shipments_list per order — that produces N+1 hammering and the "
    "model will run out of tool-call budget before finishing.\n"
    "- For any cross-domain forensic query (orders + shipments, orders + stock, etc.), first scan the "
    "available analytics_* tools — there is almost always a server-side aggregator. Only fall back to "
    "manual joining of *_list outputs when no analytics_* tool fits.\n\n"
    "LIVE SIMULATION CONTEXT:\n"
    "- The platform ships with a simulator service that continuously generates realistic supply-chain "
    "traffic (orders, shipments progressing through 15-state postal pipeline, inventory adjustments, "
    "notifications). When it is running, KPIs and counters shift between calls.\n"
    "- For forecasts, comparisons or 'right now' questions while simulation is active, always fetch "
    "FRESH data — do not assume earlier tool results are still accurate seconds later.\n"
    "- Admin-only simulator tools (visible only when role=admin):\n"
    "    * simulator_status() → {enabled, scenario, speed, uptime_secs, counters}. Use it to answer "
    "'is the simulator running?', 'what scenario is active?', 'how fast is time running?', "
    "'how many orders has the bot created today?'. Always mention the active scenario + speed when "
    "reporting live metrics.\n"
    "    * simulator_start(scenario, speed) → enables traffic generation. Default scenario='steady', "
    "speed=1.0. Valid scenarios: idle | steady | holiday_spike | carrier_failure | demand_surge. "
    "Use when user says 'start the simulator', 'enable live mode', 'turn on traffic'.\n"
    "    * simulator_stop() → pauses all actors, counters retained. Use on 'stop the simulator', "
    "'turn off live mode', 'halt traffic'.\n"
    "    * simulator_set_speed(speed) → live-adjust the multiplier without restart.\n"
    "    * simulator_set_scenario(scenario) → live-switch scenario without restart.\n"
    "- Simulator-generated orders carry the same shape as human orders — do NOT advise the user to "
    "filter them out unless they explicitly ask. They are part of the live operational state.\n"
    "- When the simulator is enabled and the user asks predictive/forecast/period-comparison questions, "
    "always fetch fresh aggregates — both sales_daily and logistics_daily are updated incrementally by "
    "the analytics-service NATS listener on every order.created / shipment_created / shipment_delivered "
    "event, so prior tool results may be seconds out of date.\n"
)

StreamCallback = Callable[[str, str], Awaitable[None]]

_GEMINI_ALLOWED_SCHEMA_KEYS = {
    "type",
    "format",
    "description",
    "title",
    "nullable",
    "enum",
    "properties",
    "required",
    "items",
    "minItems",
    "maxItems",
    "minimum",
    "maximum",
    "minLength",
    "maxLength",
    "pattern",
    "default",
    "example",
}

def _resolve_refs(node: Any, defs: dict[str, Any], _seen: tuple = ()) -> Any:

    if isinstance(node, dict):
        if "$ref" in node and isinstance(node["$ref"], str):
            ref = node["$ref"]
            key = ref.rsplit("/", 1)[-1]
            if key in _seen:
                return {"type": "object"}
            target = defs.get(key)
            if target is not None:
                resolved = _resolve_refs(target, defs, _seen + (key,))
                merged = {k: v for k, v in node.items() if k != "$ref"}
                if isinstance(resolved, dict):
                    merged = {**resolved, **merged}
                return _resolve_refs(merged, defs, _seen)
            return {k: v for k, v in node.items() if k != "$ref"}

        for combiner in ("anyOf", "oneOf"):
            if combiner in node and isinstance(node[combiner], list):
                branches = [b for b in node[combiner] if not (
                    isinstance(b, dict) and b.get("type") == "null"
                )]
                if branches:
                    chosen = _resolve_refs(branches[0], defs, _seen)
                    rest = {k: v for k, v in node.items() if k not in ("anyOf", "oneOf")}
                    if isinstance(chosen, dict):
                        return {**chosen, **rest}
                    return rest

        return {k: _resolve_refs(v, defs, _seen) for k, v in node.items()}

    if isinstance(node, list):
        return [_resolve_refs(x, defs, _seen) for x in node]

    return node

def _sanitize_schema(schema: Any, _is_property_map: bool = False) -> Any:

    if isinstance(schema, dict):
        cleaned: dict[str, Any] = {}
        for k, v in schema.items():
            if _is_property_map:

                cleaned[k] = _sanitize_schema(v, _is_property_map=False)
                continue
            if k not in _GEMINI_ALLOWED_SCHEMA_KEYS:
                continue

            child_is_property_map = (k == "properties")
            cleaned[k] = _sanitize_schema(v, _is_property_map=child_is_property_map)

        if not _is_property_map:
            if cleaned.get("type") == "array" and "items" not in cleaned:
                cleaned["items"] = {"type": "string"}

            if cleaned.get("type") == "object" and "properties" not in cleaned:
                cleaned["properties"] = {}

            if isinstance(cleaned.get("required"), list) and isinstance(cleaned.get("properties"), dict):
                valid = [r for r in cleaned["required"]
                         if isinstance(r, str) and r in cleaned["properties"]]
                if valid:
                    cleaned["required"] = valid
                else:
                    cleaned.pop("required", None)

            if "default" in cleaned and cleaned["default"] is None:
                cleaned.pop("default")

        return cleaned

    if isinstance(schema, list):
        return [_sanitize_schema(x) for x in schema]

    return schema

def build_gemini_tools(mcp_tools: list[dict[str, Any]]) -> list[types.Tool]:

    declarations = []
    for tool in mcp_tools:
        raw = dict(tool.get("parameters") or {})
        defs = (raw.get("$defs")
                or raw.get("definitions")
                or {})
        if not isinstance(defs, dict):
            defs = {}

        resolved = _resolve_refs(raw, defs)
        schema = _sanitize_schema(resolved)
        if not isinstance(schema, dict):
            schema = {"type": "object", "properties": {}}
        schema.setdefault("type", "object")
        schema.setdefault("properties", {})

        declarations.append({
            "name": tool["name"],
            "description": tool.get("description") or "",
            "parameters": schema,
        })

    return [types.Tool(function_declarations=declarations)]

async def chat_completion(
    mcp: MCPClient,
    history: list[types.Content],
    user_message: str,
    user_role: str,
    session_id: str = "",
    user_id: str | None = None,
    trace_id: str | None = None,
    plan_store: PlanStore | None = None,
    on_stream: StreamCallback | None = None,
    budget: SessionBudget | None = None,
    loop_guard: LoopGuard | None = None,
) -> tuple[str, list[types.Content]]:

    client = genai.Client(api_key=GEMINI_API_KEY)
    allowed_tools = filter_tools_by_role(mcp.tools, user_role)
    gemini_tools = build_gemini_tools(allowed_tools)

    config = types.GenerateContentConfig(
        system_instruction=(
            f"{SYSTEM_INSTRUCTION}\n\n"
            f"CONTEXT:\n"
            f"- Today's date is {datetime.now(timezone.utc).strftime('%Y-%m-%d')} (UTC).\n"
            f"- Current user role: {user_role}.\n"
        ),
        max_output_tokens=32768,
        temperature=0.4,
        tools=gemini_tools,
    )

    history = list(history)
    history.append(types.Content(role="user", parts=[types.Part(text=user_message)]))

    plan: ExecutionPlan | None = None
    if plan_store and session_id:
        plan = ExecutionPlan(session_id=session_id, intent=user_message[:500])
        await plan_store.save(plan)

    try:
        result_text = await _run_tool_loop(
            client, mcp, history, config, plan, plan_store, on_stream,
            session_id=session_id, user_id=user_id,
            trace_id=trace_id,
            budget=budget, loop_guard=loop_guard,
        )
        return result_text, history
    finally:
        if plan and plan_store:
            plan.finalize()
            await plan_store.save(plan)
            if on_stream and len(plan.steps) > 0:
                try:
                    await on_stream("plan", plan.model_dump_json())
                except Exception as exc:
                    logger.debug("plan stream failed: %s", exc)

def _is_retriable_empty_response(finish_reason: Any, has_content: bool) -> bool:

    if has_content:
        return False
    name = (getattr(finish_reason, "name", None) or str(finish_reason or "")).upper()
    return "MALFORMED_FUNCTION_CALL" in name or name == "STOP"

async def _generate_with_fallback(
    client: genai.Client,
    contents: list[types.Content],
    config: types.GenerateContentConfig,
) -> tuple[Any, Any, Any, str]:

    response = await client.aio.models.generate_content(
        model=GEMINI_MODEL, contents=contents, config=config,
    )
    candidate = (response.candidates or [None])[0]
    finish_reason = getattr(candidate, "finish_reason", None) if candidate else None
    has_content = bool(candidate and candidate.content and getattr(candidate.content, "parts", None))

    if not _is_retriable_empty_response(finish_reason, has_content):
        return response, candidate, finish_reason, GEMINI_MODEL

    if not GEMINI_FALLBACK_MODEL or GEMINI_FALLBACK_MODEL == GEMINI_MODEL:
        return response, candidate, finish_reason, GEMINI_MODEL

    logger.warning(
        "Primary model %s produced unrecoverable empty response (finish_reason=%s); retrying with %s",
        GEMINI_MODEL, finish_reason, GEMINI_FALLBACK_MODEL,
    )
    response = await client.aio.models.generate_content(
        model=GEMINI_FALLBACK_MODEL, contents=contents, config=config,
    )
    candidate = (response.candidates or [None])[0]
    finish_reason = getattr(candidate, "finish_reason", None) if candidate else None
    return response, candidate, finish_reason, GEMINI_FALLBACK_MODEL

async def _run_tool_loop(
    client: genai.Client,
    mcp: MCPClient,
    history: list[types.Content],
    config: types.GenerateContentConfig,
    plan: ExecutionPlan | None,
    plan_store: PlanStore | None,
    on_stream: StreamCallback | None,
    *,
    session_id: str = "",
    user_id: str | None = None,
    trace_id: str | None = None,
    budget: SessionBudget | None = None,
    loop_guard: LoopGuard | None = None,
) -> str:

    for round_num in range(MAX_TOOL_ROUNDS):
        logger.info("LLM round %d", round_num + 1)

        if budget and session_id:
            status = await budget.get_status(session_id, user_id)
            if status.exceeded:
                msg = (
                    f"Token budget exhausted (session used {status.session_used}/{status.session_cap}). "
                    "I cannot continue this conversation safely. Please start a new chat and narrow your request."
                )
                if on_stream:
                    await on_stream("tool_error", msg)
                    await on_stream("stream", msg)
                return msg

        with logfire_span(
            "gemini.generate_content",
            model=GEMINI_MODEL,
            session_id=session_id,
            user_id=user_id,
            trace_id=trace_id,
            round=round_num + 1,
        ) as gen_span:
            response, candidate, finish_reason, model_used = await _generate_with_fallback(
                client, history, config,
            )
            try:
                usage = getattr(response, "usage_metadata", None)
                if usage:
                    gen_span.set_attribute("input_tokens", int(getattr(usage, "prompt_token_count", 0) or 0))
                    gen_span.set_attribute("output_tokens", int(getattr(usage, "candidates_token_count", 0) or 0))
                    gen_span.set_attribute("total_tokens", int(getattr(usage, "total_token_count", 0) or 0))
                gen_span.set_attribute("model_used", model_used)
                gen_span.set_attribute("finish_reason", str(finish_reason))
            except Exception:
                pass

        if budget and session_id:
            try:
                usage = getattr(response, "usage_metadata", None)
                total = int(getattr(usage, "total_token_count", 0) or 0)
                if total > 0:
                    bs = await budget.add_usage(session_id, user_id, total)
                    if bs.exceeded:
                        logger.warning(
                            "Budget exceeded in round %d: %s (session=%d/%d)",
                            round_num + 1, bs.reason, bs.session_used, bs.session_cap,
                        )
            except Exception as exc:
                logger.debug("budget tracking failed: %s", exc)
        if model_used != GEMINI_MODEL:
            logger.info("Round %d completed via fallback model %s", round_num + 1, model_used)
        model_content = candidate.content if candidate else None

        if model_content is None or not getattr(model_content, "parts", None):
            fallback_msg = (
                "I couldn't generate a response for that request. "
                f"(finish_reason={finish_reason!r}). "
                "Please rephrase, simplify, or try a smaller scope."
            )
            logger.warning(
                "Empty model response (round %d): finish_reason=%s",
                round_num + 1, finish_reason,
            )
            if on_stream:
                await on_stream("stream", fallback_msg)
            return fallback_msg

        function_calls = []
        for part in model_content.parts:
            if part.function_call is not None:
                function_calls.append(part)

        if not function_calls:
            history.append(model_content)
            text = _extract_text(model_content)
            if on_stream:
                await on_stream("stream", text)
            return text

        logger.info("Executing %d tool call(s)", len(function_calls))
        history.append(model_content)

        function_response_parts = []
        failed_tools: list[str] = []

        for fc_part in function_calls:
            fc = fc_part.function_call
            tool_name = fc.name
            tool_args = dict(fc.args) if fc.args else {}

            if on_stream:
                await on_stream("tool_start", f"Calling {tool_name}...")

            step = None
            if plan:
                step = plan.add_step(tool=tool_name, params=tool_args)
                plan.start_step(step)
                if plan_store:
                    await plan_store.save(plan)

            loop_result = None
            if loop_guard and session_id:
                loop_result = await loop_guard.check(session_id, tool_name, tool_args)

            if loop_result and loop_result.loop:
                result = {"error": {
                    "code": "loop_detected",
                    "message": loop_result.message,
                    "suggestion": loop_result.suggestion,
                    "occurrences": loop_result.occurrences,
                }}
            else:
                entity_ids = extract_entity_ids(tool_args)
                with logfire_span(
                    "tool.call",
                    tool_name=tool_name,
                    session_id=session_id,
                    user_id=user_id,
                    trace_id=trace_id,
                    round=round_num + 1,
                    **entity_ids,
                ) as tspan:
                    result = await mcp.call_tool(tool_name, tool_args, trace_id=trace_id)
                    is_error_flag = isinstance(result, dict) and "error" in result
                    tspan.set_attribute("error", is_error_flag)
                    if is_error_flag and isinstance(result.get("error"), dict):
                        tspan.set_attribute("error_code", result["error"].get("code", ""))

            is_error = isinstance(result, dict) and "error" in result

            if step and plan:
                if is_error:
                    plan.fail_step(step, str(result["error"]))
                else:
                    plan.complete_step(step, result if isinstance(result, dict) else {"result": result})
                if plan_store:
                    await plan_store.save(plan)

            if on_stream:
                if is_error:
                    error_detail = result.get("error", "unknown error")
                    retried = result.get("retried", False)
                    suffix = f" (after {result.get('attempts', 1)} attempts)" if retried else ""
                    await on_stream("tool_error", f"{tool_name} failed: {error_detail}{suffix}")
                    failed_tools.append(tool_name)
                else:
                    await on_stream("tool_result", f"{tool_name} completed successfully")

            fr_kwargs: dict[str, Any] = {
                "name": tool_name,
                "response": {"result": result},
            }
            if fc.id:
                fr_kwargs["id"] = fc.id

            function_response_parts.append(types.Part.from_function_response(**fr_kwargs))

        if on_stream and failed_tools:
            await on_stream(
                "partial_failure",
                f"{len(failed_tools)} tool(s) failed: {', '.join(failed_tools)}. Continuing with available results.",
            )

        history.append(types.Content(role="user", parts=function_response_parts))

    history.append(types.Content(
        role="user",
        parts=[types.Part(text="Please summarize the results so far and provide your final answer.")],
    ))
    response = await client.aio.models.generate_content(
        model=GEMINI_MODEL,
        contents=history,
        config=config,
    )
    final_content = response.candidates[0].content
    history.append(final_content)
    text = _extract_text(final_content)
    if on_stream:
        await on_stream("stream", text)
    return text

def _extract_text(content: types.Content) -> str:

    if content is None or not getattr(content, "parts", None):
        return "(No text response)"
    texts = []
    for part in content.parts:
        if getattr(part, "text", None):
            texts.append(part.text)
    return "\n".join(texts) if texts else "(No text response)"
