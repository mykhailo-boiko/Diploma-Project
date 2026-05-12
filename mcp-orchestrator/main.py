
from mcp.server.fastmcp import FastMCP

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
