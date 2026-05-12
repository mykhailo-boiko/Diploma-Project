"use client";

import { useApiQuery, type SingleResponse } from "./api-hooks";

export interface AuditEvent {
  id: string;
  actor_user_id: string;
  actor_email: string;
  actor_role: string;
  service_name: string;
  action: string;
  entity_type?: string;
  entity_ids?: string[];
  params_snip?: string;
  result_status: string;
  success_count: number;
  failure_count: number;
  error_message?: string;
  trace_id?: string;
  created_at: string;
}

export interface AuditTraceResponse {
  entity_id: string;
  total: number;
  trace_ids: string[];
  events: AuditEvent[];
}

export function useAuditTrace(entityId: string | undefined, limit = 200) {
  return useApiQuery<SingleResponse<AuditTraceResponse>>(
    ["audit", "trace", entityId ?? "", String(limit)],
    `/api/v1/analytics/trace/by-entity?entity_id=${entityId ?? ""}&limit=${limit}`,
    { enabled: Boolean(entityId) },
  );
}
