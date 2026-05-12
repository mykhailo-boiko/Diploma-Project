"use client";

import { useApiQuery, useApiMutation, type SingleResponse } from "./api-hooks";

export interface SalesSummary {
  total_revenue: number;
  order_count: number;
  avg_order_value: number;
}

export interface SalesTrend {
  period: string;
  total_orders: number;
  total_revenue: number;
  avg_order_size: number;
}

export interface InventorySummary {
  total_products: number;
  total_quantity: number;
  total_reserved: number;
  total_available: number;
  low_stock_count: number;
  turnover_rate: number;
}

export interface LogisticsPerformance {
  total_shipments: number;
  delivered_count: number;
  failed_count: number;
  on_time_rate: number;
  avg_delivery_h: number;
}

export interface Anomaly {
  type: string;
  metric: string;
  value: number;
  threshold: number;
  date: string;
  severity: string;
  message: string;
}

export interface Optimization {
  product_id: string;
  product_name: string;
  product_sku: string;
  warehouse_id: string;
  current_stock: number;
  avg_daily_demand: number;
  reorder_point: number;
  recommended_order: number;
  urgency: string;
  message: string;
}

export interface ReportRequest {
  report_type: string;
  date_from: string;
  date_to: string;
}

export interface Report {
  report_type: string;
  date_from: string;
  date_to: string;
  generated_at: string;
  data: unknown;
}

function dateRangeParams(dateFrom?: string, dateTo?: string): string {
  const sp = new URLSearchParams();
  if (dateFrom) sp.set("date_from", dateFrom);
  if (dateTo) sp.set("date_to", dateTo);
  return sp.toString();
}

export function useSalesSummary(dateFrom: string, dateTo: string) {
  const qs = dateRangeParams(dateFrom, dateTo);
  return useApiQuery<{ data: SalesSummary }>(
    ["analytics", "sales-summary", dateFrom, dateTo],
    `/api/v1/analytics/sales/summary?${qs}`,
  );
}

export function useSalesTrends(dateFrom: string, dateTo: string, granularity: "day" | "week" = "day") {
  const sp = new URLSearchParams();
  sp.set("date_from", dateFrom);
  sp.set("date_to", dateTo);
  sp.set("granularity", granularity);
  return useApiQuery<{ data: SalesTrend[] }>(
    ["analytics", "sales-trends", dateFrom, dateTo, granularity],
    `/api/v1/analytics/sales/trends?${sp.toString()}`,
  );
}

export function useInventorySummary(dateFrom: string, dateTo: string) {
  const qs = dateRangeParams(dateFrom, dateTo);
  return useApiQuery<{ data: InventorySummary }>(
    ["analytics", "inventory-summary", dateFrom, dateTo],
    `/api/v1/analytics/inventory/summary?${qs}`,
  );
}

export function useLogisticsPerformance(dateFrom: string, dateTo: string) {
  const qs = dateRangeParams(dateFrom, dateTo);
  return useApiQuery<{ data: LogisticsPerformance }>(
    ["analytics", "logistics-performance", dateFrom, dateTo],
    `/api/v1/analytics/logistics/performance?${qs}`,
  );
}

export function useAnomalies(dateFrom: string, dateTo: string) {
  const qs = dateRangeParams(dateFrom, dateTo);
  return useApiQuery<{ data: Anomaly[] }>(
    ["analytics", "anomalies", dateFrom, dateTo],
    `/api/v1/analytics/anomalies?${qs}`,
  );
}

export function useOptimizations(dateFrom: string, dateTo: string) {
  const qs = dateRangeParams(dateFrom, dateTo);
  return useApiQuery<{ data: Optimization[] }>(
    ["analytics", "optimization", dateFrom, dateTo],
    `/api/v1/analytics/optimization?${qs}`,
  );
}

export function useGenerateReport() {
  return useApiMutation<SingleResponse<Report>, ReportRequest>(
    "POST",
    () => "/api/v1/analytics/report",
  );
}
