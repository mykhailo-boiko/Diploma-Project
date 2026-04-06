"""Tests for MCP tool registration and JSON Schema generation."""

from main import mcp
from mcp.server.fastmcp import FastMCP


def test_mcp_instance_created():
    """MCP server instance exists and is a FastMCP."""
    assert isinstance(mcp, FastMCP)


def test_tools_registered():
    """All expected tools are registered."""
    tools = mcp._tool_manager.list_tools()
    tool_names = {t.name for t in tools}

    expected_tools = {
        "orders_list", "orders_get", "orders_create",
        "orders_update_status", "orders_cancel", "orders_search", "orders_stats",
        "products_list", "products_get", "products_create",
        "products_update", "products_delete",
        "warehouses_list", "warehouses_get", "warehouses_create", "warehouses_update",
        "stock_list", "stock_reserve", "stock_release", "stock_adjust",
        "stock_movements", "stock_low", "stock_threshold_update",
        "inventory_report",
        "shipments_list", "shipments_get", "shipments_create",
        "shipments_update_status", "shipments_bulk_status",
        "carriers_list", "carriers_get", "carriers_create", "carriers_update",
        "routes_calculate", "logistics_performance",
        "analytics_sales", "analytics_sales_summary", "analytics_sales_trends",
        "analytics_inventory", "analytics_inventory_summary",
        "analytics_logistics", "analytics_logistics_performance",
        "analytics_anomalies", "analytics_optimization", "analytics_report",
        "notifications_list", "notifications_create", "notifications_mark_read",
        "notifications_unread_count", "notifications_preferences_get",
        "notifications_preferences_update", "notifications_bulk",
        "users_login", "users_register", "users_refresh_token",
        "users_password_reset", "users_password_reset_confirm",
        "users_me", "users_update_profile", "users_list",
        "users_create", "users_update", "users_delete",
    }

    missing = expected_tools - tool_names
    assert not missing, f"Missing tools: {missing}"
    assert len(tool_names) >= 30, f"Expected 30+ tools, got {len(tool_names)}"


def test_tool_count():
    """Verify minimum tool count per acceptance criteria (30+)."""
    tools = mcp._tool_manager.list_tools()
    assert len(tools) >= 30


def _get_tool(name: str):
    tools = mcp._tool_manager.list_tools()
    return next(t for t in tools if t.name == name)


def test_orders_list_schema():
    """Verify orders_list tool has correct JSON Schema parameters."""
    tool = _get_tool("orders_list")

    assert tool.description is not None
    assert "orders" in tool.description.lower()

    props = tool.parameters.get("properties", {})
    assert "status" in props
    assert "date_from" in props
    assert "date_to" in props
    assert "customer_name" in props
    assert "limit" in props
    assert "offset" in props


def test_orders_create_schema():
    """Verify orders_create tool has required parameters."""
    tool = _get_tool("orders_create")

    required = tool.parameters.get("required", [])
    assert "customer_name" in required
    assert "items" in required


def test_analytics_report_schema():
    """Verify analytics_report tool has correct parameters."""
    tool = _get_tool("analytics_report")

    required = tool.parameters.get("required", [])
    assert "report_type" in required
    assert "date_from" in required
    assert "date_to" in required


def test_all_tools_have_descriptions():
    """Every registered tool must have a non-empty description."""
    tools = mcp._tool_manager.list_tools()
    for tool in tools:
        assert tool.description, f"Tool {tool.name} has no description"
        assert len(tool.description) > 10, f"Tool {tool.name} description is too short"


def test_all_tools_have_parameters():
    """Every registered tool must have a parameters schema."""
    tools = mcp._tool_manager.list_tools()
    for tool in tools:
        assert tool.parameters is not None, f"Tool {tool.name} has no parameters"
        assert isinstance(tool.parameters, dict), f"Tool {tool.name} parameters is not a dict"
