"use client";

import { useApiQuery } from "./api-hooks";
import { Role } from "./types";

interface OrderStats {
  data: {
    total_orders: number;
    total_revenue: number;
    by_status: { status: string; count: number; revenue: number }[];
  };
}

interface UnreadCount {
  data: { unread_count: number };
}

interface LowStockItem {
  id: string;
  product_id: string;
  warehouse_id: string;
  quantity: number;
  reserved: number;
  available: number;
  min_threshold: number;
  product_name: string;
  product_sku: string;
}

interface LowStockResponse {
  data: LowStockItem[];
  meta: { total: number };
}

interface LogisticsPerformance {
  data: {
    total_delivered: number;
    on_time: number;
    late: number;
    on_time_rate: number;
  };
}

interface SalesSummary {
  data: {
    total_revenue: number;
    order_count: number;
    avg_order_value: number;
  };
}

interface SalesTrend {
  period: string;
  total_orders: number;
  total_revenue: number;
  avg_order_size: number;
}

interface SalesTrendsResponse {
  data: SalesTrend[];
}

interface InventorySummary {
  data: {
    total_products: number;
    total_quantity: number;
    total_reserved: number;
    total_available: number;
    low_stock_count: number;
    turnover_rate: number;
  };
}

interface AnalyticsLogisticsPerf {
  data: {
    total_shipments: number;
    delivered_count: number;
    failed_count: number;
    on_time_rate: number;
    avg_delivery_h: number;
  };
}

interface Anomaly {
  type: string;
  metric: string;
  value: number;
  threshold: number;
  date: string;
  severity: string;
  message: string;
}

interface AnomaliesResponse {
  data: Anomaly[];
}

interface Order {
  id: string;
  customer_name: string;
  status: string;
  total_amount: number;
  created_at: string;
}

interface RecentOrdersResponse {
  data: Order[];
  meta: { total: number };
}

function dateRange30d() {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 30);
  const fmt = (d: Date) => d.toISOString().slice(0, 10);
  return `date_from=${fmt(from)}&date_to=${fmt(to)}`;
}

export function useOrderStats(enabled: boolean) {
  return useApiQuery<OrderStats>(
    ["dashboard", "order-stats"],
    "/api/v1/orders/stats",
    { enabled },
  );
}

export function useUnreadCount() {
  return useApiQuery<UnreadCount>(
    ["dashboard", "unread-count"],
    "/api/v1/notifications/unread-count",
  );
}

export function useLowStock(enabled: boolean) {
  return useApiQuery<LowStockResponse>(
    ["dashboard", "low-stock"],
    "/api/v1/stock/low?limit=10",
    { enabled },
  );
}

export function useLogisticsPerformance(enabled: boolean) {
  return useApiQuery<LogisticsPerformance>(
    ["dashboard", "logistics-perf"],
    "/api/v1/logistics/performance",
    { enabled },
  );
}

export function useSalesSummary(enabled: boolean) {
  const range = dateRange30d();
  return useApiQuery<SalesSummary>(
    ["dashboard", "sales-summary"],
    `/api/v1/analytics/sales/summary?${range}`,
    { enabled },
  );
}

export function useSalesTrends(enabled: boolean) {
  const range = dateRange30d();
  return useApiQuery<SalesTrendsResponse>(
    ["dashboard", "sales-trends"],
    `/api/v1/analytics/sales/trends?${range}&granularity=day`,
    { enabled },
  );
}

export function useInventorySummary(enabled: boolean) {
  const range = dateRange30d();
  return useApiQuery<InventorySummary>(
    ["dashboard", "inventory-summary"],
    `/api/v1/analytics/inventory/summary?${range}`,
    { enabled },
  );
}

export function useAnalyticsLogisticsPerf(enabled: boolean) {
  const range = dateRange30d();
  return useApiQuery<AnalyticsLogisticsPerf>(
    ["dashboard", "analytics-logistics"],
    `/api/v1/analytics/logistics/performance?${range}`,
    { enabled },
  );
}

export function useAnomalies(enabled: boolean) {
  const range = dateRange30d();
  return useApiQuery<AnomaliesResponse>(
    ["dashboard", "anomalies"],
    `/api/v1/analytics/anomalies?${range}`,
    { enabled },
  );
}

export function useRecentOrders(enabled: boolean) {
  return useApiQuery<RecentOrdersResponse>(
    ["dashboard", "recent-orders"],
    "/api/v1/orders?limit=5&sort_by=created_at&sort_order=desc",
    { enabled },
  );
}

const ROLE_SECTIONS: Record<Role, string[]> = {
  admin: ["orders", "inventory", "logistics", "analytics", "notifications"],
  operator: ["orders", "notifications"],
  warehouse_manager: ["inventory", "notifications"],
  logistics_manager: ["logistics", "notifications"],
  analyst: ["analytics", "notifications"],
};

export function canSee(role: Role, section: string): boolean {
  return ROLE_SECTIONS[role]?.includes(section) ?? false;
}

export type {
  OrderStats,
  LowStockItem,
  SalesTrend,
  Anomaly,
  Order,
};
