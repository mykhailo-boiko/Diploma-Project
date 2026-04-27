"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";


export interface Shipment {
  id: string;
  order_id: string;
  warehouse_id: string;
  carrier_id: string;
  status: string;
  address: string;
  created_at: string;
  updated_at: string;
}


export interface ShipmentListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  status?: string;
  carrierId?: string;
  orderId?: string;
  warehouseId?: string;
  dateFrom?: string;
  dateTo?: string;
}

function buildShipmentListPath(params: ShipmentListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.status) sp.set("status", params.status);
  if (params.carrierId) sp.set("carrier_id", params.carrierId);
  if (params.orderId) sp.set("order_id", params.orderId);
  if (params.warehouseId) sp.set("warehouse_id", params.warehouseId);
  if (params.dateFrom) sp.set("date_from", params.dateFrom);
  if (params.dateTo) sp.set("date_to", params.dateTo);
  return `/api/v1/shipments?${sp.toString()}`;
}


export function useShipments(params: ShipmentListParams) {
  return useApiQuery<PaginatedResponse<Shipment>>(
    ["shipments", JSON.stringify(params)],
    buildShipmentListPath(params),
  );
}

export function useShipment(id: string) {
  return useApiQuery<SingleResponse<Shipment>>(
    ["shipments", id],
    `/api/v1/shipments/${id}`,
    { enabled: !!id },
  );
}

export function useUpdateShipmentStatus() {
  return useApiMutation<SingleResponse<Shipment>, { id: string; status: string }>(
    "PUT",
    (v) => `/api/v1/shipments/${v.id}/status`,
    { invalidateKeys: [["shipments"]] },
  );
}
