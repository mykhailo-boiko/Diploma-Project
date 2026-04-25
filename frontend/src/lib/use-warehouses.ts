"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";


export interface Warehouse {
  id: string;
  name: string;
  address?: string;
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface CreateWarehouseInput {
  name: string;
  address?: string;
}

export interface UpdateWarehouseInput {
  id: string;
  name: string;
  address?: string;
  is_active: boolean;
}


export interface WarehouseListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  name?: string;
}

function buildWarehouseListPath(params: WarehouseListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.name) sp.set("name", params.name);
  return `/api/v1/warehouses?${sp.toString()}`;
}


export function useWarehouses(params: WarehouseListParams) {
  return useApiQuery<PaginatedResponse<Warehouse>>(
    ["warehouses", JSON.stringify(params)],
    buildWarehouseListPath(params),
  );
}

export function useWarehouse(id: string) {
  return useApiQuery<SingleResponse<Warehouse>>(
    ["warehouses", id],
    `/api/v1/warehouses/${id}`,
    { enabled: !!id },
  );
}

export function useCreateWarehouse() {
  return useApiMutation<SingleResponse<Warehouse>, CreateWarehouseInput>(
    "POST",
    () => "/api/v1/warehouses",
    { invalidateKeys: [["warehouses"]] },
  );
}

export function useUpdateWarehouse() {
  return useApiMutation<SingleResponse<Warehouse>, UpdateWarehouseInput>(
    "PUT",
    (v) => `/api/v1/warehouses/${v.id}`,
    { invalidateKeys: [["warehouses"]] },
  );
}
