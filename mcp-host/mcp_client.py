
from __future__ import annotations

import asyncio
import hashlib
import json
import logging
import os
import random
from typing import Any

from mcp import ClientSession, StdioServerParameters
from mcp.client.stdio import stdio_client

from config import (
    MCP_SERVER_ARGS,
    MCP_SERVER_CMD,
    MCP_SERVER_ENV,
    RETRY_BASE_DELAY,
    RETRY_MAX_ATTEMPTS,
    RETRY_MAX_DELAY,
    TOOL_TIMEOUT,
)

logger = logging.getLogger(__name__)

class ToolTimeoutError(Exception):
    pass

class ToolRetryableError(Exception):
    pass

def _is_retryable(error: Exception) -> bool:

    msg = str(error).lower()
    retryable_patterns = [
        "connection",
        "timeout",
        "503",
        "502",
        "500",
        "unavailable",
        "broken pipe",
        "reset by peer",
        "eof",
        "network",
        "temporarily",
    ]
    return any(pattern in msg for pattern in retryable_patterns) or isinstance(
        error, (asyncio.TimeoutError, ToolTimeoutError, ToolRetryableError, ConnectionError, OSError)
    )

def _backoff_delay(attempt: int) -> float:

    delay = min(RETRY_BASE_DELAY * (2 ** attempt), RETRY_MAX_DELAY)
    jitter = random.uniform(0, delay * 0.3)
    return delay + jitter

_CACHEABLE_TOOL_PREFIXES = (
    "analytics_",
    "orders_list", "orders_get", "orders_search", "orders_stats",
    "orders_sales_by_product", "orders_customer_summary",
    "products_list", "products_get",
    "warehouses_list", "warehouses_get",
    "stock_list", "stock_low", "stock_movements",
    "shipments_list", "shipments_get",
    "carriers_list", "carriers_get",
    "notifications_list", "notifications_unread_count", "notifications_unread_counts",
    "notifications_preferences_get",
    "users_list", "users_me",
    "customers_profile_360",
    "logistics_performance",
    "inventory_report",
    "audit_query",
)

_MUTATION_TOOL_HINTS = (
    "_create", "_update", "_delete", "_cancel",
    "_reserve", "_release", "_adjust",
    "_bulk", "_reassign", "_mark_read", "_register", "_login",
    "_password_reset",
)

_PER_TOOL_TTL: dict[str, int] = {
    "analytics_anomalies": 60,
    "analytics_carriers_performance": 60,
    "analytics_quick_cancellations": 60,
    "analytics_rebalancing_recommendations": 60,
    "analytics_sales_summary": 30,
    "analytics_sales_trends": 30,
    "analytics_period_comparison": 60,
    "customers_profile_360": 60,
    "audit_query": 15,
    "users_me": 30,
    "simulator_status": 0,
}

_DEFAULT_CACHE_TTL = 30

def _is_cacheable(name: str) -> bool:
    if any(hint in name for hint in _MUTATION_TOOL_HINTS):
        return False
    return any(name == p or name.startswith(p) for p in _CACHEABLE_TOOL_PREFIXES)

def _is_mutation(name: str) -> bool:
    return any(hint in name for hint in _MUTATION_TOOL_HINTS)

def _ttl_for(name: str) -> int:
    return _PER_TOOL_TTL.get(name, _DEFAULT_CACHE_TTL)

def _cache_key(name: str, arguments: dict[str, Any]) -> str:
    canonical = json.dumps(arguments, sort_keys=True, default=str)
    digest = hashlib.sha256(f"{name}|{canonical}".encode()).hexdigest()[:32]
    return f"toolcache:{digest}"

class MCPClient:

    def __init__(self, redis: Any | None = None) -> None:
        self._session: ClientSession | None = None
        self._tools: list[dict[str, Any]] = []
        self._stdio_context: Any = None
        self._session_context: Any = None
        self._read: Any = None
        self._write: Any = None
        self._redis = redis
        self._cache_hits = 0
        self._cache_misses = 0

    def attach_cache(self, redis_client: Any) -> None:
        self._redis = redis_client

    async def connect(self) -> None:

        env = {**os.environ, **MCP_SERVER_ENV}
        server_params = StdioServerParameters(
            command=MCP_SERVER_CMD,
            args=list(MCP_SERVER_ARGS),
            env=env,
        )

        self._stdio_context = stdio_client(server_params)
        self._read, self._write = await self._stdio_context.__aenter__()

        self._session_context = ClientSession(self._read, self._write)
        self._session = await self._session_context.__aenter__()

        await self._session.initialize()

        tools_result = await self._session.list_tools()
        self._tools = []
        for tool in tools_result.tools:
            schema = tool.inputSchema if tool.inputSchema else {"type": "object", "properties": {}}
            self._tools.append({
                "name": tool.name,
                "description": tool.description or "",
                "parameters": schema,
            })

        logger.info("MCP Client connected — %d tools available", len(self._tools))

    async def close(self) -> None:

        if self._session_context is not None:
            try:
                await self._session_context.__aexit__(None, None, None)
            except Exception:
                logger.debug("Session close error (ignored)")
        if self._stdio_context is not None:
            try:
                await self._stdio_context.__aexit__(None, None, None)
            except Exception:
                logger.debug("Stdio close error (ignored)")
        self._session = None
        logger.info("MCP Client disconnected")

    @property
    def tools(self) -> list[dict[str, Any]]:

        return list(self._tools)

    async def call_tool(
        self,
        name: str,
        arguments: dict[str, Any],
        timeout: int | None = None,
        trace_id: str | None = None,
    ) -> Any:

        if self._session is None:
            return {"error": "MCP session not connected"}

        _ = trace_id  # propagated for telemetry only; stdio MCP does not forward headers

        if self._redis is not None and _is_cacheable(name):
            cached = await self._cache_get(name, arguments)
            if cached is not None:
                self._cache_hits += 1
                logger.info("cache HIT %s (total hits=%d, misses=%d)", name, self._cache_hits, self._cache_misses)
                return cached
            self._cache_misses += 1

        effective_timeout = timeout if timeout is not None else TOOL_TIMEOUT
        last_error: Exception | None = None

        for attempt in range(RETRY_MAX_ATTEMPTS):
            if attempt > 0:
                delay = _backoff_delay(attempt - 1)
                logger.info("Retry %d/%d for tool %s (delay %.1fs)", attempt + 1, RETRY_MAX_ATTEMPTS, name, delay)
                await asyncio.sleep(delay)

            logger.info("Calling MCP tool: %s(%s)", name, json.dumps(arguments, default=str)[:200])

            try:
                result = await asyncio.wait_for(
                    self._session.call_tool(name, arguments),
                    timeout=effective_timeout,
                )

                texts = []
                for block in result.content:
                    if hasattr(block, "text"):
                        texts.append(block.text)

                combined = "\n".join(texts)

                try:
                    parsed = json.loads(combined)
                except (json.JSONDecodeError, ValueError):
                    parsed = {"result": combined}

                if self._redis is not None:
                    if _is_cacheable(name) and not (isinstance(parsed, dict) and "error" in parsed):
                        await self._cache_set(name, arguments, parsed)
                    elif _is_mutation(name) and not (isinstance(parsed, dict) and "error" in parsed):
                        await self._cache_invalidate_all()

                return parsed

            except asyncio.TimeoutError:
                last_error = ToolTimeoutError(f"Tool '{name}' timed out after {effective_timeout}s")
                logger.warning("Tool call timed out: %s (attempt %d/%d)", name, attempt + 1, RETRY_MAX_ATTEMPTS)

            except Exception as exc:
                last_error = exc
                if not _is_retryable(exc):
                    logger.error("Non-retryable tool call failure: %s — %s", name, exc)
                    return {"error": str(exc), "retryable": False}
                logger.warning(
                    "Retryable tool call failure: %s — %s (attempt %d/%d)",
                    name, exc, attempt + 1, RETRY_MAX_ATTEMPTS,
                )

        error_msg = str(last_error) if last_error else "Unknown error"
        logger.error("Tool call failed after %d attempts: %s — %s", RETRY_MAX_ATTEMPTS, name, error_msg)
        return {"error": error_msg, "retried": True, "attempts": RETRY_MAX_ATTEMPTS}

    async def _cache_get(self, name: str, arguments: dict[str, Any]) -> Any | None:
        try:
            raw = await self._redis.get(_cache_key(name, arguments))
            if raw is None:
                return None
            return json.loads(raw)
        except Exception as exc:
            logger.debug("cache get failed for %s: %s", name, exc)
            return None

    async def _cache_set(self, name: str, arguments: dict[str, Any], value: Any) -> None:
        ttl = _ttl_for(name)
        if ttl <= 0:
            return
        try:
            await self._redis.set(
                _cache_key(name, arguments),
                json.dumps(value, default=str),
                ex=ttl,
            )
        except Exception as exc:
            logger.debug("cache set failed for %s: %s", name, exc)

    async def _cache_invalidate_all(self) -> None:
        try:
            cursor = 0
            deleted = 0
            while True:
                cursor, keys = await self._redis.scan(cursor=cursor, match="toolcache:*", count=500)
                if keys:
                    await self._redis.delete(*keys)
                    deleted += len(keys)
                if cursor == 0:
                    break
            if deleted > 0:
                logger.info("cache INVALIDATED %d entries after mutation", deleted)
        except Exception as exc:
            logger.debug("cache invalidation failed: %s", exc)
