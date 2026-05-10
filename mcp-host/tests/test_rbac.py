"""Tests for RBAC tool filtering."""

from rbac import ROLE_PERMISSIONS, filter_tools_by_role

_ALL_TOOLS = [
    {"name": "orders_list", "description": "List orders", "parameters": {}},
    {"name": "orders_get", "description": "Get order", "parameters": {}},
    {"name": "orders_create", "description": "Create order", "parameters": {}},
    {"name": "products_list", "description": "List products", "parameters": {}},
    {"name": "products_create", "description": "Create product", "parameters": {}},
    {"name": "warehouses_list", "description": "List warehouses", "parameters": {}},
    {"name": "stock_list", "description": "List stock", "parameters": {}},
    {"name": "stock_reserve", "description": "Reserve stock", "parameters": {}},
    {"name": "inventory_report", "description": "Inventory report", "parameters": {}},
    {"name": "shipments_list", "description": "List shipments", "parameters": {}},
    {"name": "shipments_create", "description": "Create shipment", "parameters": {}},
    {"name": "carriers_list", "description": "List carriers", "parameters": {}},
    {"name": "routes_calculate", "description": "Calculate route", "parameters": {}},
    {"name": "logistics_performance", "description": "Performance", "parameters": {}},
    {"name": "analytics_sales", "description": "Sales data", "parameters": {}},
    {"name": "analytics_anomalies", "description": "Anomalies", "parameters": {}},
    {"name": "notifications_list", "description": "List notifications", "parameters": {}},
    {"name": "notifications_create", "description": "Create notification", "parameters": {}},
    {"name": "users_login", "description": "Login", "parameters": {}},
    {"name": "users_me", "description": "My profile", "parameters": {}},
    {"name": "users_update_profile", "description": "Update profile", "parameters": {}},
    {"name": "users_list", "description": "List users (admin)", "parameters": {}},
    {"name": "users_create", "description": "Create user (admin)", "parameters": {}},
    {"name": "users_delete", "description": "Delete user (admin)", "parameters": {}},
    {"name": "users_refresh_token", "description": "Refresh token", "parameters": {}},
    {"name": "users_password_reset", "description": "Password reset", "parameters": {}},
]


def _tool_names(tools):
    return {t["name"] for t in tools}


class TestAdminRole:
    def test_admin_sees_all_tools(self):
        result = filter_tools_by_role(_ALL_TOOLS, "admin")
        assert len(result) == len(_ALL_TOOLS)

    def test_admin_wildcard_in_permissions(self):
        assert "*" in ROLE_PERMISSIONS["admin"]


class TestOperatorRole:
    def test_operator_sees_orders_and_notifications(self):
        result = filter_tools_by_role(_ALL_TOOLS, "operator")
        names = _tool_names(result)
        assert "orders_list" in names
        assert "orders_create" in names
        assert "notifications_list" in names
        assert "notifications_create" in names

    def test_operator_cannot_see_inventory(self):
        result = filter_tools_by_role(_ALL_TOOLS, "operator")
        names = _tool_names(result)
        assert "products_list" not in names
        assert "stock_reserve" not in names
        assert "inventory_report" not in names

    def test_operator_cannot_see_logistics(self):
        result = filter_tools_by_role(_ALL_TOOLS, "operator")
        names = _tool_names(result)
        assert "shipments_list" not in names
        assert "carriers_list" not in names

    def test_operator_cannot_see_analytics(self):
        result = filter_tools_by_role(_ALL_TOOLS, "operator")
        names = _tool_names(result)
        assert "analytics_sales" not in names

    def test_operator_sees_common_profile_tools(self):
        result = filter_tools_by_role(_ALL_TOOLS, "operator")
        names = _tool_names(result)
        assert "users_me" in names
        assert "users_update_profile" in names

    def test_operator_does_not_see_preauth_tools(self):
        result = filter_tools_by_role(_ALL_TOOLS, "operator")
        names = _tool_names(result)
        assert "users_login" not in names
        assert "users_refresh_token" not in names
        assert "users_password_reset" not in names

    def test_operator_cannot_see_admin_user_tools(self):
        result = filter_tools_by_role(_ALL_TOOLS, "operator")
        names = _tool_names(result)
        assert "users_list" not in names
        assert "users_create" not in names
        assert "users_delete" not in names


class TestWarehouseManagerRole:
    def test_sees_inventory_and_orders(self):
        result = filter_tools_by_role(_ALL_TOOLS, "warehouse_manager")
        names = _tool_names(result)
        assert "products_list" in names
        assert "stock_reserve" in names
        assert "inventory_report" in names
        assert "orders_list" in names

    def test_cannot_see_logistics(self):
        result = filter_tools_by_role(_ALL_TOOLS, "warehouse_manager")
        names = _tool_names(result)
        assert "shipments_list" not in names
        assert "carriers_list" not in names


class TestLogisticsManagerRole:
    def test_sees_logistics_and_orders(self):
        result = filter_tools_by_role(_ALL_TOOLS, "logistics_manager")
        names = _tool_names(result)
        assert "shipments_list" in names
        assert "carriers_list" in names
        assert "routes_calculate" in names
        assert "logistics_performance" in names
        assert "orders_list" in names

    def test_cannot_see_inventory(self):
        result = filter_tools_by_role(_ALL_TOOLS, "logistics_manager")
        names = _tool_names(result)
        assert "products_list" not in names
        assert "stock_reserve" not in names


class TestAnalystRole:
    def test_sees_analytics(self):
        result = filter_tools_by_role(_ALL_TOOLS, "analyst")
        names = _tool_names(result)
        assert "analytics_sales" in names
        assert "analytics_anomalies" in names

    def test_cannot_see_orders_or_inventory(self):
        result = filter_tools_by_role(_ALL_TOOLS, "analyst")
        names = _tool_names(result)
        assert "orders_list" not in names
        assert "products_list" not in names

    def test_analyst_only_sees_analytics_and_profile(self):
        result = filter_tools_by_role(_ALL_TOOLS, "analyst")
        names = _tool_names(result)
        for name in names:
            assert name.startswith("analytics_") or name in {"users_me", "users_update_profile"}, (
                f"unexpected tool leaked to analyst: {name}"
            )


class TestUnknownRole:
    def test_unknown_role_gets_only_common_tools(self):
        result = filter_tools_by_role(_ALL_TOOLS, "unknown_role")
        names = _tool_names(result)
        assert "users_me" in names
        assert "users_update_profile" in names
        assert "users_login" not in names
        assert "orders_list" not in names
        assert "products_list" not in names
        assert "analytics_sales" not in names

    def test_empty_tools_returns_empty(self):
        result = filter_tools_by_role([], "admin")
        assert result == []
