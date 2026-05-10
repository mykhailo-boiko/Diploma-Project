"""Gemini LLM integration with MCP tool execution loop, streaming, and execution plan tracking."""

from __future__ import annotations

import logging
from collections.abc import Awaitable, Callable
from datetime import datetime, timezone
from typing import Any

from google import genai
from google.genai import types

from config import GEMINI_API_KEY, GEMINI_FALLBACK_MODEL, GEMINI_MODEL, MAX_TOOL_ROUNDS
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
    "OPERATIONAL FORENSICS — ALWAYS PREFER DEDICATED TOOLS OVER CLIENT-SIDE LOOPS:\n"
    "- 'orders cancelled within X minutes/hour after shipped', 'quick cancel after dispatch', "
    "'shipping-handover failure', 'cancellation patterns by carrier' — use analytics_quick_cancellations. "
    "It joins orders × shipments × carriers server-side and groups by carrier × destination city. "
    "DO NOT iterate orders_list + shipments_list per order — that produces N+1 hammering and the "
    "model will run out of tool-call budget before finishing.\n"
    "- For any cross-domain forensic query (orders + shipments, orders + stock, etc.), first scan the "
    "available analytics_* tools — there is almost always a server-side aggregator. Only fall back to "
    "manual joining of *_list outputs when no analytics_* tool fits.\n"
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
    """Inline-resolve $ref / $defs / definitions references so the schema is
    self-contained (Gemini does not understand $ref/$defs)."""
    if isinstance(node, dict):
        if "$ref" in node and isinstance(node["$ref"], str):
            ref = node["$ref"]
            key = ref.rsplit("/", 1)[-1]
            if key in _seen:
                return {"type": "object"}  # break recursion
            target = defs.get(key)
            if target is not None:
                resolved = _resolve_refs(target, defs, _seen + (key,))
                merged = {k: v for k, v in node.items() if k != "$ref"}
                if isinstance(resolved, dict):
                    merged = {**resolved, **merged}
                return _resolve_refs(merged, defs, _seen)
            return {k: v for k, v in node.items() if k != "$ref"}

        # Collapse anyOf/oneOf into the first non-null branch.
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
    """Recursively strip JSON-Schema fields Gemini does not understand.

    `_is_property_map=True` when the dict represents a `properties` map (where
    keys are field names, not schema keywords) so we don't filter keys against
    the whitelist there.
    """
    if isinstance(schema, dict):
        cleaned: dict[str, Any] = {}
        for k, v in schema.items():
            if _is_property_map:
                # Keep all keys; recurse into each value as a schema node.
                cleaned[k] = _sanitize_schema(v, _is_property_map=False)
                continue
            if k not in _GEMINI_ALLOWED_SCHEMA_KEYS:
                continue
            # `properties` value is itself a property map — names are arbitrary.
            child_is_property_map = (k == "properties")
            cleaned[k] = _sanitize_schema(v, _is_property_map=child_is_property_map)

        if not _is_property_map:
            if cleaned.get("type") == "array" and "items" not in cleaned:
                cleaned["items"] = {"type": "string"}

            if cleaned.get("type") == "object" and "properties" not in cleaned:
                cleaned["properties"] = {}

            # Drop required entries that don't reference existing properties
            if isinstance(cleaned.get("required"), list) and isinstance(cleaned.get("properties"), dict):
                valid = [r for r in cleaned["required"]
                         if isinstance(r, str) and r in cleaned["properties"]]
                if valid:
                    cleaned["required"] = valid
                else:
                    cleaned.pop("required", None)

            # `default: null` is invalid for Gemini's schema validator.
            if "default" in cleaned and cleaned["default"] is None:
                cleaned.pop("default")

        return cleaned

    if isinstance(schema, list):
        return [_sanitize_schema(x) for x in schema]

    return schema


def build_gemini_tools(mcp_tools: list[dict[str, Any]]) -> list[types.Tool]:
    """Convert MCP tool definitions to Gemini function declarations.

    Pipeline: $defs/$ref inline → drop unknown keys → enforce Gemini quirks.
    """
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
    plan_store: PlanStore | None = None,
    on_stream: StreamCallback | None = None,
) -> tuple[str, list[types.Content]]:
    """Run a full chat turn: send user message, execute tool calls, return final text.

    When plan_store is provided, creates an ExecutionPlan that tracks every tool call
    with timing, inputs, outputs, and status.

    When on_stream is provided, sends incremental updates to the client:
      - on_stream("tool_start", "Calling orders_list...")
      - on_stream("tool_result", "orders_list completed successfully")
      - on_stream("tool_error", "orders_list failed: timeout after 30s")
      - on_stream("stream", "partial text chunk")

    Returns (response_text, updated_history).
    """
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
        result_text = await _run_tool_loop(client, mcp, history, config, plan, plan_store, on_stream)
        return result_text, history
    finally:
        if plan and plan_store:
            plan.finalize()
            await plan_store.save(plan)


def _is_retriable_empty_response(finish_reason: Any, has_content: bool) -> bool:
    """Conditions where a retry on the fallback model is likely to help.

    MALFORMED_FUNCTION_CALL: known flash glitch on parallel/complex tool calls — pro is more reliable.
    STOP with empty parts: model exhausted output budget on internal thoughts before producing text — pro
    has stronger reasoning compaction and a larger output budget headroom.
    """
    if has_content:
        return False
    name = (getattr(finish_reason, "name", None) or str(finish_reason or "")).upper()
    return "MALFORMED_FUNCTION_CALL" in name or name == "STOP"


async def _generate_with_fallback(
    client: genai.Client,
    contents: list[types.Content],
    config: types.GenerateContentConfig,
) -> tuple[Any, Any, Any, str]:
    """Generate content with the primary model; on MALFORMED_FUNCTION_CALL retry once with the fallback model."""
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
) -> str:
    """Execute the LLM tool-calling loop with streaming and partial failure handling."""
    for round_num in range(MAX_TOOL_ROUNDS):
        logger.info("LLM round %d", round_num + 1)

        response, candidate, finish_reason, model_used = await _generate_with_fallback(
            client, history, config,
        )
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

            result = await mcp.call_tool(tool_name, tool_args)

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
    """Extract all text parts from a Content object."""
    if content is None or not getattr(content, "parts", None):
        return "(No text response)"
    texts = []
    for part in content.parts:
        if getattr(part, "text", None):
            texts.append(part.text)
    return "\n".join(texts) if texts else "(No text response)"
