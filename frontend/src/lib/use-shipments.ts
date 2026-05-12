"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";


export interface AddressDetails {
  full_name?: string;
  phone?: string;
  email?: string;
  company?: string;
  street?: string;
  city?: string;
  region?: string;
  postcode?: string;
  country?: string;
  delivery_notes?: string;
}

export interface Shipment {
  id: string;
  order_id: string;
  warehouse_id: string;
  carrier_id: string;
  status: string;
  tracking_number: string;
  address: string;
  recipient: AddressDetails;
  sender: AddressDetails;
  estimated_delivery_at?: string;
  delivered_at?: string;
  delivery_attempts: number;
  delivery_signature?: string;
  delivery_photo_url?: string;
  current_location_city?: string;
  current_location_hub?: string;
  created_at: string;
  updated_at: string;
}

export interface ShipmentEvent {
  id: string;
  event_type: string;
  location_city?: string;
  location_hub?: string;
  notes?: string;
  occurred_at: string;
  recorded_by: string;
}

export interface DeliveryAttempt {
  id: string;
  attempt_number: number;
  reason: string;
  notes?: string;
  next_attempt_at?: string;
  occurred_at: string;
}

export interface Timeline {
  shipment: Shipment;
  events: ShipmentEvent[];
  delivery_attempts: DeliveryAttempt[];
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

export function useShipmentTimeline(id: string) {
  return useApiQuery<SingleResponse<Timeline>>(
    ["shipments", id, "timeline"],
    `/api/v1/shipments/${id}/timeline`,
    { enabled: !!id },
  );
}

export function useUpdateRecipient() {
  return useApiMutation<SingleResponse<Shipment>, { id: string; patch: Partial<AddressDetails> }>(
    "PATCH",
    (v) => `/api/v1/shipments/${v.id}/recipient`,
    {
      invalidateKeys: [["shipments"]],
      bodyFn: (v) => v.patch,
    },
  );
}

export function useShipmentRecordDelivery() {
  return useApiMutation<
    SingleResponse<Shipment>,
    { id: string; signature_name: string; photo_url?: string }
  >(
    "POST",
    (v) => `/api/v1/shipments/${v.id}/record-delivery`,
    {
      invalidateKeys: [["shipments"]],
      bodyFn: (v) => ({ signature_name: v.signature_name, photo_url: v.photo_url || "" }),
    },
  );
}
