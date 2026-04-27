"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";


export interface Carrier {
  id: string;
  name: string;
  type: string;
  cost_per_km: number;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateCarrierInput {
  name: string;
  type: string;
  cost_per_km: number;
}

export interface UpdateCarrierInput {
  name: string;
  type: string;
  cost_per_km: number;
  is_active: boolean;
}


export interface CarrierListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  type?: string;
  name?: string;
}

function buildCarrierListPath(params: CarrierListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.type) sp.set("type", params.type);
  if (params.name) sp.set("name", params.name);
  return `/api/v1/carriers?${sp.toString()}`;
}


export function useCarriers(params: CarrierListParams) {
  return useApiQuery<PaginatedResponse<Carrier>>(
    ["carriers", JSON.stringify(params)],
    buildCarrierListPath(params),
  );
}

export function useCarrier(id: string) {
  return useApiQuery<SingleResponse<Carrier>>(
    ["carriers", id],
    `/api/v1/carriers/${id}`,
    { enabled: !!id },
  );
}

export function useCreateCarrier() {
  return useApiMutation<SingleResponse<Carrier>, CreateCarrierInput>(
    "POST",
    () => "/api/v1/carriers",
    { invalidateKeys: [["carriers"]] },
  );
}

export function useUpdateCarrier() {
  return useApiMutation<SingleResponse<Carrier>, { id: string } & UpdateCarrierInput>(
    "PUT",
    (v) => `/api/v1/carriers/${v.id}`,
    { invalidateKeys: [["carriers"]] },
  );
}
