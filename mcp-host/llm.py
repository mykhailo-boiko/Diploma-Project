"""Gemini LLM integration with MCP tool execution loop, streaming, and execution plan tracking."""

from __future__ import annotations

import logging
from collections.abc import Awaitable, Callable
from typing import Any

from google import genai
from google.genai import types

from config import GEMINI_API_KEY, GEMINI_MODEL, MAX_TOOL_ROUNDS
from mcp_client import MCPClient
from models import ExecutionPlan
from plan_store import PlanStore
from rbac import filter_tools_by_role

logger = logging.getLogger(__name__)

SYSTEM_INSTRUCTION = (
    "You are an AI assistant for a supply chain management platform called ChainOrchestra. "
    "You help users manage orders, inventory, logistics, analytics, notifications, and user accounts. "
    "Use the available tools to fetch data and perform actions. "
    "Always provide clear, concise answers based on the tool results. "
    "If a tool call fails, explain the error to the user and suggest alternatives. "
    "When performing destructive actions (delete, cancel), confirm with the user first. "
    "Format responses in a readable way — use lists and tables where appropriate."
)

StreamCallback = Callable[[str, str], Awaitable[None]]


def build_gemini_tools(mcp_tools: list[dict[str, Any]]) -> list[types.Tool]:
    """Convert MCP tool definitions to Gemini function declarations."""
    declarations = []
    for tool in mcp_tools:
        schema = dict(tool["parameters"])
        schema.setdefault("type", "object")
        schema.setdefault("properties", {})

        declarations.append({
            "name": tool["name"],
            "description": tool["description"],
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
        system_instruction=f"{SYSTEM_INSTRUCTION}\n\nCurrent user role: {user_role}.",
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

        response = await client.aio.models.generate_content(
            model=GEMINI_MODEL,
            contents=history,
            config=config,
        )

        candidate = response.candidates[0]
        model_content = candidate.content

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
    texts = []
    for part in content.parts:
        if part.text:
            texts.append(part.text)
    return "\n".join(texts) if texts else "(No text response)"
