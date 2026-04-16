"""Tests for LLM integration module."""

from llm import SYSTEM_INSTRUCTION, build_gemini_tools


class TestBuildGeminiTools:
    def test_converts_mcp_tools(self):
        mcp_tools = [
            {
                "name": "orders_list",
                "description": "List orders",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "status": {"type": "string", "description": "Order status"},
                        "limit": {"type": "integer", "description": "Max results"},
                    },
                },
            },
            {
                "name": "orders_get",
                "description": "Get order by ID",
                "parameters": {
                    "type": "object",
                    "properties": {
                        "order_id": {"type": "string", "description": "Order ID"},
                    },
                    "required": ["order_id"],
                },
            },
        ]

        tools = build_gemini_tools(mcp_tools)
        assert len(tools) == 1

        tool = tools[0]
        assert tool.function_declarations is not None
        assert len(tool.function_declarations) == 2

    def test_empty_tools(self):
        tools = build_gemini_tools([])
        assert len(tools) == 1
        assert len(tools[0].function_declarations) == 0

    def test_tool_with_no_properties(self):
        mcp_tools = [
            {
                "name": "stats",
                "description": "Get stats",
                "parameters": {},
            },
        ]
        tools = build_gemini_tools(mcp_tools)
        assert len(tools[0].function_declarations) == 1

    def test_system_instruction_content(self):
        assert "supply chain" in SYSTEM_INSTRUCTION.lower()
        assert "ChainOrchestra" in SYSTEM_INSTRUCTION
