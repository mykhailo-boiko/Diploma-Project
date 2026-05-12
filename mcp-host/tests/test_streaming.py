
from unittest.mock import AsyncMock, MagicMock, patch

import pytest

from llm import _extract_text, chat_completion

def _mock_content(text: str | None = None, function_calls: list | None = None):

    parts = []
    if text:
        part = MagicMock()
        part.function_call = None
        part.text = text
        parts.append(part)
    if function_calls:
        for fc in function_calls:
            part = MagicMock()
            part.function_call = MagicMock()
            part.function_call.name = fc["name"]
            part.function_call.args = fc.get("args", {})
            part.function_call.id = fc.get("id")
            part.text = None
            parts.append(part)
    content = MagicMock()
    content.parts = parts
    return content

def _mock_response(content):

    candidate = MagicMock()
    candidate.content = content
    response = MagicMock()
    response.candidates = [candidate]
    return response

class TestStreamCallback:
    @pytest.mark.asyncio
    async def test_stream_called_on_final_text(self):

        on_stream = AsyncMock()
        mock_mcp = MagicMock()
        mock_mcp.tools = [{"name": "t", "description": "d", "parameters": {"type": "object", "properties": {}}}]

        final_content = _mock_content(text="Here are your orders.")
        mock_response = _mock_response(final_content)

        with patch("llm.genai") as mock_genai,\
             patch("llm.filter_tools_by_role", return_value=mock_mcp.tools):
            mock_client = MagicMock()
            mock_genai.Client.return_value = mock_client
            mock_client.aio.models.generate_content = AsyncMock(return_value=mock_response)

            result, _ = await chat_completion(
                mcp=mock_mcp,
                history=[],
                user_message="List orders",
                user_role="admin",
                on_stream=on_stream,
            )

        assert result == "Here are your orders."
        on_stream.assert_any_call("stream", "Here are your orders.")

    @pytest.mark.asyncio
    async def test_stream_tool_start_and_result(self):

        on_stream = AsyncMock()
        mock_mcp = MagicMock()
        mock_mcp.tools = [
            {"name": "orders_list", "description": "List orders", "parameters": {"type": "object", "properties": {}}},
        ]
        mock_mcp.call_tool = AsyncMock(return_value={"orders": []})

        fc_content = _mock_content(function_calls=[{"name": "orders_list", "args": {}, "id": None}])
        final_content = _mock_content(text="No orders found.")

        with patch("llm.genai") as mock_genai,\
             patch("llm.filter_tools_by_role", return_value=mock_mcp.tools):
            mock_client = MagicMock()
            mock_genai.Client.return_value = mock_client
            mock_client.aio.models.generate_content = AsyncMock(
                side_effect=[_mock_response(fc_content), _mock_response(final_content)]
            )

            await chat_completion(
                mcp=mock_mcp,
                history=[],
                user_message="List orders",
                user_role="admin",
                on_stream=on_stream,
            )

        on_stream.assert_any_call("tool_start", "Calling orders_list...")
        on_stream.assert_any_call("tool_result", "orders_list completed successfully")

    @pytest.mark.asyncio
    async def test_stream_tool_error_and_partial_failure(self):

        on_stream = AsyncMock()
        mock_mcp = MagicMock()
        mock_mcp.tools = [
            {"name": "orders_list", "description": "d", "parameters": {"type": "object", "properties": {}}},
            {"name": "inventory_list", "description": "d", "parameters": {"type": "object", "properties": {}}},
        ]
        mock_mcp.call_tool = AsyncMock(side_effect=[
            {"orders": [{"id": "1"}]},
            {"error": "connection refused", "retried": True, "attempts": 3},
        ])

        fc_content = _mock_content(function_calls=[
            {"name": "orders_list", "args": {}, "id": None},
            {"name": "inventory_list", "args": {}, "id": None},
        ])
        final_content = _mock_content(text="Orders found, but inventory unavailable.")

        with patch("llm.genai") as mock_genai,\
             patch("llm.filter_tools_by_role", return_value=mock_mcp.tools):
            mock_client = MagicMock()
            mock_genai.Client.return_value = mock_client
            mock_client.aio.models.generate_content = AsyncMock(
                side_effect=[_mock_response(fc_content), _mock_response(final_content)]
            )

            await chat_completion(
                mcp=mock_mcp,
                history=[],
                user_message="Show orders and inventory",
                user_role="admin",
                on_stream=on_stream,
            )

        on_stream.assert_any_call("tool_result", "orders_list completed successfully")
        on_stream.assert_any_call("tool_error", "inventory_list failed: connection refused (after 3 attempts)")
        on_stream.assert_any_call(
            "partial_failure",
            "1 tool(s) failed: inventory_list. Continuing with available results.",
        )

    @pytest.mark.asyncio
    async def test_no_stream_when_callback_is_none(self):

        mock_mcp = MagicMock()
        mock_mcp.tools = []

        final_content = _mock_content(text="Hello.")
        mock_response = _mock_response(final_content)

        with patch("llm.genai") as mock_genai,\
             patch("llm.filter_tools_by_role", return_value=[]):
            mock_client = MagicMock()
            mock_genai.Client.return_value = mock_client
            mock_client.aio.models.generate_content = AsyncMock(return_value=mock_response)

            result, _ = await chat_completion(
                mcp=mock_mcp,
                history=[],
                user_message="Hi",
                user_role="admin",
                on_stream=None,
            )

        assert result == "Hello."

class TestExtractText:
    def test_single_text_part(self):
        content = _mock_content(text="Hello")
        assert _extract_text(content) == "Hello"

    def test_no_text_parts(self):
        content = MagicMock()
        part = MagicMock()
        part.text = None
        content.parts = [part]
        assert _extract_text(content) == "(No text response)"

    def test_empty_text(self):
        content = MagicMock()
        part = MagicMock()
        part.text = ""
        content.parts = [part]
        assert _extract_text(content) == "(No text response)"
