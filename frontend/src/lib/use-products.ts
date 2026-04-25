"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";


export interface Product {
  id: string;
  sku: string;
  name: string;
  description?: string;
  category?: string;
  unit_price: number;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
}

export interface CreateProductInput {
  sku: string;
  name: string;
  description?: string;
  category?: string;
  unit_price: number;
}

export interface UpdateProductInput {
  id: string;
  name: string;
  description?: string;
  category?: string;
  unit_price: number;
}


export interface ProductListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  sku?: string;
  name?: string;
  category?: string;
}

function buildProductListPath(params: ProductListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.sku) sp.set("sku", params.sku);
  if (params.name) sp.set("name", params.name);
  if (params.category) sp.set("category", params.category);
  return `/api/v1/products?${sp.toString()}`;
}


export function useProducts(params: ProductListParams) {
  return useApiQuery<PaginatedResponse<Product>>(
    ["products", JSON.stringify(params)],
    buildProductListPath(params),
  );
}

export function useProduct(id: string) {
  return useApiQuery<SingleResponse<Product>>(
    ["products", id],
    `/api/v1/products/${id}`,
    { enabled: !!id },
  );
}

export function useCreateProduct() {
  return useApiMutation<SingleResponse<Product>, CreateProductInput>(
    "POST",
    () => "/api/v1/products",
    { invalidateKeys: [["products"]] },
  );
}

export function useUpdateProduct() {
  return useApiMutation<SingleResponse<Product>, UpdateProductInput>(
    "PUT",
    (v) => `/api/v1/products/${v.id}`,
    { invalidateKeys: [["products"]] },
  );
}

export function useDeleteProduct() {
  return useApiMutation<void, { id: string }>(
    "DELETE",
    (v) => `/api/v1/products/${v.id}`,
    { invalidateKeys: [["products"]] },
  );
}
