"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";


export interface Stock {
  id: string;
  product_id: string;
  warehouse_id: string;
  quantity: number;
  reserved: number;
  available: number;
  min_threshold: number;
  updated_at: string;
}

export interface StockMovement {
  id: string;
  stock_id: string;
  product_id: string;
  warehouse_id: string;
  type: string;
  quantity: number;
  reference: string;
  created_at: string;
}

export interface LowStockItem {
  id: string;
  product_id: string;
  warehouse_id: string;
  quantity: number;
  reserved: number;
  available: number;
  min_threshold: number;
  updated_at: string;
  product_name: string;
  product_sku: string;
}

export interface InventoryReport {
  total_products: number;
  total_warehouses: number;
  total_quantity: number;
  total_reserved: number;
  total_available: number;
  low_stock_count: number;
  by_warehouse: { warehouse_id: string; warehouse_name: string; total_quantity: number; total_reserved: number; total_available: number; product_count: number }[];
  by_category: { category: string; total_quantity: number; total_reserved: number; total_available: number; product_count: number }[];
}


export interface StockListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  productId?: string;
  warehouseId?: string;
}

export interface MovementListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  productId?: string;
  warehouseId?: string;
  type?: string;
}

function buildStockListPath(params: StockListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.productId) sp.set("product_id", params.productId);
  if (params.warehouseId) sp.set("warehouse_id", params.warehouseId);
  return `/api/v1/stock?${sp.toString()}`;
}

function buildMovementListPath(params: MovementListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.productId) sp.set("product_id", params.productId);
  if (params.warehouseId) sp.set("warehouse_id", params.warehouseId);
  if (params.type) sp.set("type", params.type);
  return `/api/v1/stock/movements?${sp.toString()}`;
}


export function useStock(params: StockListParams) {
  return useApiQuery<PaginatedResponse<Stock>>(
    ["stock", JSON.stringify(params)],
    buildStockListPath(params),
  );
}

export function useStockMovements(params: MovementListParams) {
  return useApiQuery<PaginatedResponse<StockMovement>>(
    ["stock-movements", JSON.stringify(params)],
    buildMovementListPath(params),
  );
}

export function useLowStock() {
  return useApiQuery<PaginatedResponse<LowStockItem>>(
    ["stock", "low"],
    "/api/v1/stock/low?limit=100&offset=0",
  );
}

export function useInventoryReport() {
  return useApiQuery<SingleResponse<InventoryReport>>(
    ["inventory", "report"],
    "/api/v1/inventory/report",
  );
}

export function useReserveStock() {
  return useApiMutation<SingleResponse<Stock>, { product_id: string; warehouse_id: string; quantity: number; reference?: string }>(
    "POST",
    () => "/api/v1/stock/reserve",
    { invalidateKeys: [["stock"], ["stock-movements"]] },
  );
}

export function useReleaseStock() {
  return useApiMutation<SingleResponse<Stock>, { product_id: string; warehouse_id: string; quantity: number; reference?: string }>(
    "POST",
    () => "/api/v1/stock/release",
    { invalidateKeys: [["stock"], ["stock-movements"]] },
  );
}

export function useAdjustStock() {
  return useApiMutation<SingleResponse<Stock>, { product_id: string; warehouse_id: string; quantity: number; type: string; reference?: string }>(
    "POST",
    () => "/api/v1/stock/adjust",
    { invalidateKeys: [["stock"], ["stock-movements"]] },
  );
}

export function useUpdateThreshold() {
  return useApiMutation<SingleResponse<Stock>, { product_id: string; warehouse_id: string; threshold: number }>(
    "PUT",
    () => "/api/v1/stock/threshold",
    { invalidateKeys: [["stock"], ["stock", "low"]] },
  );
}
