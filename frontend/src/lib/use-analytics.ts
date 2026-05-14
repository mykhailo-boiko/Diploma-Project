"use client";

import { useMutation } from "@tanstack/react-query";
import { useApiQuery, useApiMutation, type SingleResponse } from "./api-hooks";
import { ApiError } from "./api";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

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
  category?: string;
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
  product_sku: string;
  product_name: string;
  warehouse_id: string;
  warehouse_name: string;
  current_stock: number;
  min_threshold: number;
  reorder_point: number;
  recommended_order: number;
  avg_daily_demand: number;
  days_until_stockout: number;
  urgency: string;
  reason: string;
}

export interface ReportRequest {
  report_type: string;
  date_from: string;
  date_to: string;
  format?: "json" | "csv";
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

export interface DownloadReportResult {
  filename: string;
  size: number;
  contentType: string;
}

export function useDownloadReport() {
  return useMutation<DownloadReportResult, ApiError, ReportRequest>({
    mutationFn: async (req) => {
      const token =
        typeof window !== "undefined"
          ? localStorage.getItem("access_token")
          : null;
      const headers: Record<string, string> = {
        "Content-Type": "application/json",
      };
      if (token) headers["Authorization"] = `Bearer ${token}`;

      const res = await fetch(`${API_BASE}/api/v1/analytics/report`, {
        method: "POST",
        headers,
        body: JSON.stringify(req),
      });

      if (res.status === 401) {
        if (typeof window !== "undefined") {
          localStorage.removeItem("access_token");
          localStorage.removeItem("refresh_token");
          window.location.href = "/login";
        }
        throw new ApiError(401, "unauthorized", "Unauthorized");
      }

      const contentType = res.headers.get("Content-Type") ?? "";

      if (!res.ok) {
        let code = "unknown";
        let message = res.statusText;
        if (contentType.includes("application/json")) {
          const errBody = await res.json().catch(() => ({}));
          code = errBody?.error?.code ?? code;
          message = errBody?.error?.message ?? message;
        } else {
          message = (await res.text().catch(() => message)) || message;
        }
        throw new ApiError(res.status, code, message);
      }

      const headerName =
        res.headers.get("X-Report-Filename") ??
        parseFilenameFromDisposition(res.headers.get("Content-Disposition"));
      const fallback = `chainorchestra-${req.report_type}-${req.date_from}-to-${req.date_to}.${
        req.format === "csv" ? "csv" : "json"
      }`;
      const filename = headerName || fallback;

      const blob = await res.blob();
      triggerBlobDownload(blob, filename);

      return { filename, size: blob.size, contentType };
    },
  });
}

function parseFilenameFromDisposition(value: string | null): string | null {
  if (!value) return null;
  const match = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(value);
  return match ? decodeURIComponent(match[1]) : null;
}

function triggerBlobDownload(blob: Blob, filename: string) {
  if (typeof window === "undefined") return;
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = filename;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  setTimeout(() => URL.revokeObjectURL(url), 1000);
}
