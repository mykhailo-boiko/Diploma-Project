"""Tests for MCP Client module."""

import pytest

from mcp_client import MCPClient


class TestMCPClient:
    def test_initial_state(self):
        client = MCPClient()
        assert client._session is None
        assert client.tools == []

    @pytest.mark.asyncio
    async def test_call_tool_without_connection(self):
        client = MCPClient()
        result = await client.call_tool("test_tool", {"arg": "value"})
        assert "error" in result
        assert "not connected" in result["error"]

    def test_tools_returns_copy(self):
        client = MCPClient()
        client._tools = [{"name": "test"}]
        tools = client.tools
        tools.append({"name": "extra"})
        assert len(client._tools) == 1
