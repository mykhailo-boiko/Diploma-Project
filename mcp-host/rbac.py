
from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)

_ORDERS_PREFIXES = ("orders_",)
_INVENTORY_PREFIXES = ("products_", "warehouses_", "stock_", "inventory_")
_LOGISTICS_PREFIXES = ("shipments_", "carriers_", "routes_", "logistics_")
_ANALYTICS_PREFIXES = ("analytics_", "customers_profile_360")
_NOTIFICATIONS_PREFIXES = ("notifications_",)
_COMMON_PREFIXES = ("users_me", "users_update_profile")

_ADMIN_ONLY_PREFIXES: tuple[str, ...] = (
    "audit_",
    "simulator_",
    "users_register",
    "users_delete_user",
    "users_admin_",
    "_set_trace_context",
)

ROLE_PERMISSIONS: dict[str, tuple[str, ...]] = {
    "admin": ("*",),
    "operator": _ORDERS_PREFIXES + _NOTIFICATIONS_PREFIXES,
    "warehouse_manager": _INVENTORY_PREFIXES + _ORDERS_PREFIXES,
    "logistics_manager": _LOGISTICS_PREFIXES + _ORDERS_PREFIXES,
    "analyst": _ANALYTICS_PREFIXES,
}

def _is_common_tool(name: str) -> bool:

    return any(name == prefix or name.startswith(prefix) for prefix in _COMMON_PREFIXES)

def _is_admin_only(name: str) -> bool:

    return any(name == prefix or name.startswith(prefix) for prefix in _ADMIN_ONLY_PREFIXES)

def is_tool_allowed(name: str, user_role: str) -> bool:

    if user_role == "admin":
        return True
    if _is_admin_only(name):
        return False
    prefixes = ROLE_PERMISSIONS.get(user_role) or ()
    if _is_common_tool(name):
        return True
    return any(name.startswith(p) for p in prefixes)

def filter_tools_by_role(
    tools: list[dict[str, Any]],
    user_role: str,
) -> list[dict[str, Any]]:

    if user_role == "admin":
        return tools

    filtered = [tool for tool in tools if is_tool_allowed(tool["name"], user_role)]

    logger.info(
        "RBAC filter: role=%s, total=%d, allowed=%d",
        user_role, len(tools), len(filtered),
    )
    return filtered
