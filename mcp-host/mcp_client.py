"""MCP Client — connects to the MCP Server via stdio subprocess."""

from __future__ import annotations

import asyncio
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
    """Raised when a tool call exceeds the configured timeout."""


class ToolRetryableError(Exception):
    """Raised for transient errors that may succeed on retry."""


def _is_retryable(error: Exception) -> bool:
    """Determine if an error is transient and worth retrying."""
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
    """Calculate exponential backoff delay with jitter."""
    delay = min(RETRY_BASE_DELAY * (2 ** attempt), RETRY_MAX_DELAY)
    jitter = random.uniform(0, delay * 0.3)
    return delay + jitter


class MCPClient:
    """Manages a persistent connection to the MCP Server subprocess."""

    def __init__(self) -> None:
        self._session: ClientSession | None = None
        self._tools: list[dict[str, Any]] = []
        self._stdio_context: Any = None
        self._session_context: Any = None
        self._read: Any = None
        self._write: Any = None

    async def connect(self) -> None:
        """Start the MCP Server subprocess and initialize the session."""
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
        """Shut down the MCP session and subprocess."""
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
        """Return cached tool definitions."""
        return list(self._tools)

    async def call_tool(
        self,
        name: str,
        arguments: dict[str, Any],
        timeout: int | None = None,
    ) -> Any:
        """Execute a tool via MCP with timeout and retry on transient failures.

        Returns the parsed JSON content on success, or an error dict on failure.
        """
        if self._session is None:
            return {"error": "MCP session not connected"}

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
                    return json.loads(combined)
                except (json.JSONDecodeError, ValueError):
                    return {"result": combined}

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
