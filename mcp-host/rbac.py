
from __future__ import annotations

import logging
from typing import Any

logger = logging.getLogger(__name__)

_ORDERS_PREFIXES = ("orders_",)
_INVENTORY_PREFIXES = ("products_", "warehouses_", "stock_", "inventory_")
_LOGISTICS_PREFIXES = ("shipments_", "carriers_", "routes_", "logistics_")
_ANALYTICS_PREFIXES = ("analytics_",)
_NOTIFICATIONS_PREFIXES = ("notifications_",)
_COMMON_PREFIXES = ("users_me", "users_update_profile")

ROLE_PERMISSIONS: dict[str, tuple[str, ...]] = {
    "admin": ("*",),
    "operator": _ORDERS_PREFIXES + _NOTIFICATIONS_PREFIXES,
    "warehouse_manager": _INVENTORY_PREFIXES + _ORDERS_PREFIXES,
    "logistics_manager": _LOGISTICS_PREFIXES + _ORDERS_PREFIXES,
    "analyst": _ANALYTICS_PREFIXES,
}

def _is_common_tool(name: str) -> bool:

    return any(name == prefix or name.startswith(prefix) for prefix in _COMMON_PREFIXES)

def filter_tools_by_role(
    tools: list[dict[str, Any]],
    user_role: str,
) -> list[dict[str, Any]]:

    prefixes = ROLE_PERMISSIONS.get(user_role)

    if prefixes and "*" in prefixes:
        return tools

    allowed_prefixes = prefixes or ()
    filtered = []
    for tool in tools:
        name: str = tool["name"]
        if _is_common_tool(name):
            filtered.append(tool)
        elif any(name.startswith(p) for p in allowed_prefixes):
            filtered.append(tool)

    logger.info(
        "RBAC filter: role=%s, total=%d, allowed=%d",
        user_role, len(tools), len(filtered),
    )
    return filtered
