import { Role } from "./types";

export interface NavItem {
  label: string;
  href: string;
  icon: string;
  roles: Role[];
}

export const navItems: NavItem[] = [
  {
    label: "Dashboard",
    href: "/dashboard",
    icon: "LayoutDashboard",
    roles: [
      "admin",
      "operator",
      "warehouse_manager",
      "logistics_manager",
      "analyst",
    ],
  },
  {
    label: "Orders",
    href: "/orders",
    icon: "ShoppingCart",
    roles: ["admin", "operator"],
  },
  {
    label: "Products",
    href: "/products",
    icon: "Package",
    roles: ["admin", "warehouse_manager"],
  },
  {
    label: "Stock",
    href: "/stock",
    icon: "Warehouse",
    roles: ["admin", "warehouse_manager"],
  },
  {
    label: "Warehouses",
    href: "/warehouses",
    icon: "Building2",
    roles: ["admin", "warehouse_manager"],
  },
  {
    label: "Shipments",
    href: "/shipments",
    icon: "Truck",
    roles: ["admin", "logistics_manager"],
  },
  {
    label: "Carriers",
    href: "/carriers",
    icon: "Ship",
    roles: ["admin", "logistics_manager"],
  },
  {
    label: "Analytics",
    href: "/analytics",
    icon: "BarChart3",
    roles: ["admin", "analyst"],
  },
  {
    label: "Notifications",
    href: "/notifications",
    icon: "Bell",
    roles: [
      "admin",
      "operator",
      "warehouse_manager",
      "logistics_manager",
      "analyst",
    ],
  },
  {
    label: "Users",
    href: "/users",
    icon: "Users",
    roles: ["admin"],
  },
  {
    label: "Simulator",
    href: "/admin/simulator",
    icon: "Activity",
    roles: ["admin"],
  },
];

export function getNavItemsForRole(role: Role): NavItem[] {
  return navItems.filter((item) => item.roles.includes(role));
}

export function formatRole(role: Role): string {
  return role
    .split("_")
    .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
    .join(" ");
}
