"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";
import type { Role } from "./types";

export interface UserFull {
  id: string;
  email: string;
  first_name: string;
  last_name: string;
  role: Role;
  created_at: string;
  updated_at: string;
  deleted_at?: string;
}

export interface CreateUserInput {
  email: string;
  password: string;
  first_name: string;
  last_name: string;
  role: Role;
}

export interface UpdateUserInput {
  id: string;
  email?: string;
  first_name?: string;
  last_name?: string;
  role?: Role;
}

export interface UserListParams {
  page: number;
  pageSize: number;
  sortField?: string;
  sortDesc?: boolean;
  role?: string;
  email?: string;
  name?: string;
}

function buildUserListPath(params: UserListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.sortField) {
    sp.set("sort", params.sortField);
    sp.set("order", params.sortDesc ? "desc" : "asc");
  }
  if (params.role) sp.set("role", params.role);
  if (params.email) sp.set("email", params.email);
  if (params.name) sp.set("name", params.name);
  return `/api/v1/users?${sp.toString()}`;
}

export function useUsers(params: UserListParams) {
  return useApiQuery<PaginatedResponse<UserFull>>(
    ["users", JSON.stringify(params)],
    buildUserListPath(params),
  );
}

export function useCreateUser() {
  return useApiMutation<SingleResponse<UserFull>, CreateUserInput>(
    "POST",
    () => "/api/v1/users",
    { invalidateKeys: [["users"]] },
  );
}

export function useUpdateUser() {
  return useApiMutation<SingleResponse<UserFull>, UpdateUserInput>(
    "PUT",
    (v) => `/api/v1/users/${v.id}`,
    { invalidateKeys: [["users"]] },
  );
}

export function useDeleteUser() {
  return useApiMutation<void, { id: string }>(
    "DELETE",
    (v) => `/api/v1/users/${v.id}`,
    { invalidateKeys: [["users"]] },
  );
}
