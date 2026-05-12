"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";

export interface OrderItem {
  id: string;
  order_id: string;
  product_id: string;
  name: string;
  quantity: number;
  unit_price: number;
  subtotal: number;
}

export interface Order {
  id: string;
  customer_name: string;
  status: string;
  total_amount: number;
  cancel_reason?: string;
  items?: OrderItem[];
  created_at: string;
  updated_at: string;
}

export interface OrderStats {
  total_orders: number;
  total_revenue: number;
  by_status: { status: string; count: number; revenue: number }[];
}

export interface CreateOrderInput {
  customer_name: string;
  items: { product_id: string; name: string; quantity: number; unit_price: number }[];
}

export interface OrderListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  status?: string;
  dateFrom?: string;
  dateTo?: string;
  customerName?: string;
}

function buildOrderListPath(params: OrderListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.status) sp.set("status", params.status);
  if (params.dateFrom) sp.set("date_from", params.dateFrom);
  if (params.dateTo) sp.set("date_to", params.dateTo);
  if (params.customerName) sp.set("customer_name", params.customerName);
  return `/api/v1/orders?${sp.toString()}`;
}

export function useOrders(params: OrderListParams) {
  return useApiQuery<PaginatedResponse<Order>>(
    ["orders", JSON.stringify(params)],
    buildOrderListPath(params),
  );
}

export function useOrder(id: string) {
  return useApiQuery<SingleResponse<Order>>(
    ["orders", id],
    `/api/v1/orders/${id}`,
    { enabled: !!id },
  );
}

export function useOrderSearch(query: string, page: number, pageSize: number) {
  const sp = new URLSearchParams();
  sp.set("q", query);
  sp.set("limit", String(pageSize));
  sp.set("offset", String((page - 1) * pageSize));
  return useApiQuery<PaginatedResponse<Order>>(
    ["orders", "search", query, String(page)],
    `/api/v1/orders/search?${sp.toString()}`,
    { enabled: query.length >= 2 },
  );
}

export function useCreateOrder() {
  return useApiMutation<SingleResponse<Order>, CreateOrderInput>(
    "POST",
    () => "/api/v1/orders",
    { invalidateKeys: [["orders"]] },
  );
}

export function useUpdateOrderStatus() {
  return useApiMutation<SingleResponse<Order>, { id: string; status: string }>(
    "PUT",
    (v) => `/api/v1/orders/${v.id}/status`,
    { invalidateKeys: [["orders"]] },
  );
}

export function useCancelOrder() {
  return useApiMutation<SingleResponse<Order>, { id: string; reason: string }>(
    "POST",
    (v) => `/api/v1/orders/${v.id}/cancel`,
    { invalidateKeys: [["orders"]] },
  );
}
