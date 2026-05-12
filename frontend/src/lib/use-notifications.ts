"use client";

import { useApiQuery, useApiMutation, type PaginatedResponse, type SingleResponse } from "./api-hooks";

export interface Notification {
  id: string;
  user_id: string;
  type: string;
  title: string;
  message: string;
  status: string;
  read_at?: string;
  created_at: string;
}

export interface NotificationPreference {
  id: string;
  user_id: string;
  type: string;
  in_app: boolean;
  email: boolean;
  sms: boolean;
  updated_at: string;
}

export interface UnreadCount {
  unread_count: number;
}

export interface CreateNotificationInput {
  user_id: string;
  type: string;
  title: string;
  message: string;
}

export interface BulkNotificationInput {
  user_ids: string[];
  type: string;
  title: string;
  message: string;
}

export interface UpdatePreferenceInput {
  type: string;
  in_app: boolean;
  email: boolean;
  sms: boolean;
}

export interface NotificationListParams {
  page: number;
  pageSize: number;
  type?: string;
}

function buildNotificationListPath(params: NotificationListParams): string {
  const sp = new URLSearchParams();
  sp.set("limit", String(params.pageSize));
  sp.set("offset", String((params.page - 1) * params.pageSize));
  if (params.type) sp.set("type", params.type);
  return `/api/v1/notifications?${sp.toString()}`;
}

export function useNotifications(params: NotificationListParams) {
  return useApiQuery<PaginatedResponse<Notification>>(
    ["notifications", JSON.stringify(params)],
    buildNotificationListPath(params),
  );
}

export function useUnreadCount() {
  return useApiQuery<{ data: UnreadCount }>(
    ["notifications", "unread-count"],
    "/api/v1/notifications/unread-count",
  );
}

export function useMarkAsRead() {
  return useApiMutation<SingleResponse<Notification>, { id: string }>(
    "PUT",
    (v) => `/api/v1/notifications/${v.id}/read`,
    { invalidateKeys: [["notifications"]] },
  );
}

export function useNotificationPreferences() {
  return useApiQuery<{ data: NotificationPreference[] }>(
    ["notifications", "preferences"],
    "/api/v1/notifications/preferences",
  );
}

export function useUpdatePreference() {
  return useApiMutation<SingleResponse<NotificationPreference>, UpdatePreferenceInput>(
    "PUT",
    () => "/api/v1/notifications/preferences",
    { invalidateKeys: [["notifications", "preferences"]] },
  );
}

export function useCreateNotification() {
  return useApiMutation<SingleResponse<Notification>, CreateNotificationInput>(
    "POST",
    () => "/api/v1/notifications",
    { invalidateKeys: [["notifications"]] },
  );
}

export function useBulkNotification() {
  return useApiMutation<{ data: { total: number; success: number; failed: number } }, BulkNotificationInput>(
    "POST",
    () => "/api/v1/notifications/bulk",
    { invalidateKeys: [["notifications"]] },
  );
}
