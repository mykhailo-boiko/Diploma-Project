
import asyncio
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from mcp_client import MCPClient, ToolTimeoutError, _backoff_delay, _is_retryable

class TestIsRetryable:
    def test_timeout_error(self):
        assert _is_retryable(asyncio.TimeoutError()) is True

    def test_tool_timeout_error(self):
        assert _is_retryable(ToolTimeoutError("timed out")) is True

    def test_connection_error(self):
        assert _is_retryable(ConnectionError("refused")) is True

    def test_os_error(self):
        assert _is_retryable(OSError("network unreachable")) is True

    def test_message_based_503(self):
        assert _is_retryable(Exception("service returned 503")) is True

    def test_message_based_500(self):
        assert _is_retryable(Exception("server error 500")) is True

    def test_message_based_connection(self):
        assert _is_retryable(Exception("connection reset")) is True

    def test_message_based_timeout(self):
        assert _is_retryable(Exception("request timeout")) is True

    def test_non_retryable_value_error(self):
        assert _is_retryable(ValueError("invalid param")) is False

    def test_non_retryable_key_error(self):
        assert _is_retryable(KeyError("missing_key")) is False

class TestBackoffDelay:
    def test_first_attempt(self):
        delay = _backoff_delay(0)
        assert 1.0 <= delay <= 1.4

    def test_second_attempt(self):
        delay = _backoff_delay(1)
        assert 2.0 <= delay <= 2.7

    def test_capped_at_max(self):
        delay = _backoff_delay(10)
        assert delay <= 14.0

class TestCallToolTimeout:
    @pytest.mark.asyncio
    async def test_timeout_triggers_retry(self):
        client = MCPClient()
        session = AsyncMock()
        client._session = session

        result_mock = MagicMock()
        text_block = MagicMock()
        text_block.text = '{"data": "ok"}'
        result_mock.content = [text_block]

        session.call_tool = AsyncMock(side_effect=[
            asyncio.TimeoutError(),
            result_mock,
        ])

        with patch("mcp_client.TOOL_TIMEOUT", 1),\
             patch("mcp_client.RETRY_BASE_DELAY", 0.01),\
             patch("mcp_client.RETRY_MAX_DELAY", 0.05):
            result = await client.call_tool("test_tool", {})

        assert result == {"data": "ok"}
        assert session.call_tool.call_count == 2

    @pytest.mark.asyncio
    async def test_all_retries_exhausted(self):
        client = MCPClient()
        session = AsyncMock()
        client._session = session

        session.call_tool = AsyncMock(side_effect=asyncio.TimeoutError())

        with patch("mcp_client.TOOL_TIMEOUT", 1),\
             patch("mcp_client.RETRY_MAX_ATTEMPTS", 2),\
             patch("mcp_client.RETRY_BASE_DELAY", 0.01),\
             patch("mcp_client.RETRY_MAX_DELAY", 0.05):
            result = await client.call_tool("test_tool", {})

        assert "error" in result
        assert "timed out" in result["error"]
        assert result["retried"] is True
        assert result["attempts"] == 2

    @pytest.mark.asyncio
    async def test_non_retryable_error_no_retry(self):
        client = MCPClient()
        session = AsyncMock()
        client._session = session

        session.call_tool = AsyncMock(side_effect=ValueError("bad param"))

        with patch("mcp_client.RETRY_MAX_ATTEMPTS", 3):
            result = await client.call_tool("test_tool", {})

        assert "error" in result
        assert result["retryable"] is False
        assert session.call_tool.call_count == 1

    @pytest.mark.asyncio
    async def test_custom_timeout(self):
        client = MCPClient()
        session = AsyncMock()
        client._session = session

        async def slow_call(*args, **kwargs):
            await asyncio.sleep(0.2)

        session.call_tool = AsyncMock(side_effect=slow_call)

        with patch("mcp_client.RETRY_MAX_ATTEMPTS", 1),\
             patch("mcp_client.RETRY_BASE_DELAY", 0.01):
            result = await client.call_tool("test_tool", {}, timeout=0.05)

        assert "error" in result
        assert "timed out" in result["error"]

    @pytest.mark.asyncio
    async def test_retryable_connection_error(self):
        client = MCPClient()
        session = AsyncMock()
        client._session = session

        result_mock = MagicMock()
        text_block = MagicMock()
        text_block.text = '{"ok": true}'
        result_mock.content = [text_block]

        session.call_tool = AsyncMock(side_effect=[
            ConnectionError("refused"),
            result_mock,
        ])

        with patch("mcp_client.RETRY_BASE_DELAY", 0.01),\
             patch("mcp_client.RETRY_MAX_DELAY", 0.05):
            result = await client.call_tool("test_tool", {})

        assert result == {"ok": True}
        assert session.call_tool.call_count == 2

    @pytest.mark.asyncio
    async def test_success_on_first_try(self):
        client = MCPClient()
        session = AsyncMock()
        client._session = session

        result_mock = MagicMock()
        text_block = MagicMock()
        text_block.text = '{"success": true}'
        result_mock.content = [text_block]
        session.call_tool = AsyncMock(return_value=result_mock)

        result = await client.call_tool("test_tool", {"arg": "val"})

        assert result == {"success": True}
        assert session.call_tool.call_count == 1
