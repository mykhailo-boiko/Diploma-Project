
from mcp.server.fastmcp import FastMCP

from http_client import set_trace_id
from tools import (
    analytics_tools,
    inventory_tools,
    logistics_tools,
    notifications_tools,
    orders_tools,
    simulator_tools,
    users_tools,
)

mcp = FastMCP(
    "Supply Chain MCP Server",
    instructions=(
        "You are a supply chain management assistant. "
        "Use the available tools to manage orders, inventory, logistics, "
        "analytics, notifications, and users. "
        "Always confirm destructive actions before proceeding."
    ),
)

@mcp.tool(description="Internal control tool used by the host to attach a trace_id to subsequent HTTP calls. Not for direct LLM use. Args: trace_id: trace identifier string or empty to clear.")
async def _set_trace_context(trace_id: str) -> dict[str, str]:
    if trace_id:
        set_trace_id(trace_id)
    else:
        set_trace_id(None)
    return {"status": "ok"}

orders_tools.register(mcp)
inventory_tools.register(mcp)
logistics_tools.register(mcp)
analytics_tools.register(mcp)
notifications_tools.register(mcp)
users_tools.register(mcp)
simulator_tools.register(mcp)

def main() -> None:

    mcp.run()

if __name__ == "__main__":
    main()
