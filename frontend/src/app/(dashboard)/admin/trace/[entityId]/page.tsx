"use client";

import { use } from "react";
import { useAuditTrace, type AuditEvent } from "@/lib/use-audit-trace";

type Tone = "neutral" | "positive" | "warning" | "danger";

const actionTone = (e: AuditEvent): Tone => {
  if (e.result_status === "failed") return "danger";
  if (e.result_status === "partial") return "warning";
  if (e.action.endsWith(".create")) return "positive";
  if (e.action.endsWith(".cancel") || e.action.includes("delete")) return "warning";
  return "neutral";
};

const toneClass: Record<Tone, string> = {
  neutral: "border-gray-200 bg-white",
  positive: "border-emerald-200 bg-emerald-50",
  warning: "border-amber-200 bg-amber-50",
  danger: "border-red-200 bg-red-50",
};

export default function TraceViewerPage({
  params,
}: {
  params: Promise<{ entityId: string }>;
}) {
  const { entityId } = use(params);
  const query = useAuditTrace(entityId);
  const data = query.data?.data;

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold text-gray-900">Entity audit trace</h1>
        <p className="text-sm text-gray-600">
          Full ordered timeline of all write operations that touched this entity ID, with actor,
          service, params snapshot, and correlation trace IDs for cross-referencing with Logfire.
        </p>
        <p className="text-xs text-gray-500">
          entity_id: <code className="rounded bg-gray-100 px-1.5 py-0.5">{entityId}</code>
        </p>
      </header>

      {query.isLoading && <div className="text-sm text-gray-500">Loading…</div>}
      {query.error && (
        <div className="rounded-md border border-red-200 bg-red-50 p-4 text-sm text-red-800">
          Failed to load trace: {query.error.message}
        </div>
      )}

      {data && (
        <>
          <section className="grid gap-3 md:grid-cols-3">
            <KPI label="Total events" value={data.total} />
            <KPI label="Distinct trace IDs" value={data.trace_ids?.length ?? 0} />
            <KPI
              label="Services involved"
              value={new Set(data.events.map((e) => e.service_name)).size}
            />
          </section>

          {data.trace_ids?.length > 0 && (
            <section className="rounded-md border border-gray-200 bg-white p-4">
              <h2 className="mb-2 text-sm font-semibold text-gray-700">Trace IDs (Logfire deep-links)</h2>
              <div className="flex flex-wrap gap-2 text-xs">
                {data.trace_ids.map((tid) => (
                  <code key={tid} className="rounded bg-gray-100 px-2 py-1 font-mono text-gray-700">
                    {tid}
                  </code>
                ))}
              </div>
            </section>
          )}

          <section className="space-y-2">
            <h2 className="text-sm font-semibold text-gray-700">Timeline</h2>
            {data.events.length === 0 ? (
              <div className="rounded-md border border-gray-200 bg-white p-4 text-sm text-gray-500">
                No audit events recorded for this entity.
              </div>
            ) : (
              <ol className="space-y-2">
                {data.events.map((ev) => (
                  <li key={ev.id} className={`rounded-md border p-3 text-xs ${toneClass[actionTone(ev)]}`}>
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <div className="flex items-center gap-2">
                        <span className="font-semibold text-gray-900">{ev.action}</span>
                        <span className="rounded bg-white px-1.5 py-0.5 font-mono text-[10px] text-gray-600">
                          {ev.service_name}
                        </span>
                        <span className="text-[10px] uppercase tracking-wide text-gray-500">
                          {ev.result_status}
                        </span>
                      </div>
                      <span className="text-[10px] text-gray-500">
                        {new Date(ev.created_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="mt-1 text-gray-700">
                      <span className="font-medium">{ev.actor_email}</span>{" "}
                      <span className="text-gray-500">({ev.actor_role})</span>
                    </div>
                    {ev.entity_ids && ev.entity_ids.length > 0 && (
                      <div className="mt-1 truncate text-[10px] text-gray-500">
                        entities: {ev.entity_ids.join(", ")}
                      </div>
                    )}
                    {ev.params_snip && (
                      <pre className="mt-2 max-h-32 overflow-auto whitespace-pre-wrap rounded bg-gray-900/5 p-2 text-[10px] text-gray-700">
                        {ev.params_snip}
                      </pre>
                    )}
                    {ev.error_message && (
                      <div className="mt-2 rounded bg-red-100 px-2 py-1 text-[10px] text-red-800">
                        {ev.error_message}
                      </div>
                    )}
                    {ev.trace_id && (
                      <div className="mt-1 text-[10px] text-gray-500">
                        trace_id: <code className="font-mono">{ev.trace_id}</code>
                      </div>
                    )}
                  </li>
                ))}
              </ol>
            )}
          </section>
        </>
      )}
    </div>
  );
}

function KPI({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
      <div className="text-xs font-medium uppercase tracking-wide text-gray-500">{label}</div>
      <div className="mt-1 text-xl font-semibold text-gray-900">{value}</div>
    </div>
  );
}
