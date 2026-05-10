"use client";

import { useState } from "react";
import { ChevronDown, ChevronRight, CheckCircle2, XCircle, Loader2, Clock } from "lucide-react";
import clsx from "clsx";
import type { ExecutionPlan, PlanStep } from "@/lib/use-chat-socket";

interface PlanPanelProps {
  plan: ExecutionPlan;
}

function StepIcon({ status }: { status: PlanStep["status"] }) {
  if (status === "success") {
    return <CheckCircle2 className="h-4 w-4 text-emerald-600" />;
  }
  if (status === "failed") {
    return <XCircle className="h-4 w-4 text-red-600" />;
  }
  if (status === "running") {
    return <Loader2 className="h-4 w-4 animate-spin text-blue-600" />;
  }
  return <Clock className="h-4 w-4 text-gray-400" />;
}

function formatDuration(ms: number | null): string {
  if (ms == null) return "—";
  if (ms < 1000) return `${ms}ms`;
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatParams(params: Record<string, unknown>): string {
  const keys = Object.keys(params);
  if (keys.length === 0) return "—";
  return keys
    .map((k) => {
      const v = params[k];
      const formatted =
        typeof v === "string"
          ? v.length > 30
            ? `"${v.slice(0, 27)}..."`
            : `"${v}"`
          : typeof v === "object"
            ? Array.isArray(v)
              ? `[${(v as unknown[]).length} items]`
              : "{...}"
            : String(v);
      return `${k}=${formatted}`;
    })
    .join(", ");
}

function StepRow({ step, index }: { step: PlanStep; index: number }) {
  const [open, setOpen] = useState(false);
  const paramsBrief = formatParams(step.params);
  const statusBadgeColor =
    step.status === "success"
      ? "text-emerald-700 bg-emerald-50 ring-emerald-100"
      : step.status === "failed"
        ? "text-red-700 bg-red-50 ring-red-100"
        : step.status === "running"
          ? "text-blue-700 bg-blue-50 ring-blue-100"
          : "text-gray-700 bg-gray-50 ring-gray-100";

  return (
    <div className="border-t border-gray-100 first:border-t-0">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-start gap-3 px-3 py-2 text-left hover:bg-gray-50"
      >
        <div className="flex-shrink-0 pt-0.5">
          {open ? (
            <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 text-gray-400" />
          )}
        </div>
        <div className="flex-shrink-0 pt-0.5">
          <StepIcon status={step.status} />
        </div>
        <div className="flex-shrink-0 pt-0.5 text-xs font-semibold text-gray-500">
          {index + 1}
        </div>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <code className="text-sm font-medium text-gray-900">{step.tool}</code>
            <span
              className={clsx(
                "rounded-full px-1.5 py-0.5 text-[10px] font-medium ring-1",
                statusBadgeColor,
              )}
            >
              {step.status}
            </span>
            <span className="ml-auto text-xs tabular-nums text-gray-500">
              {formatDuration(step.duration_ms)}
            </span>
          </div>
          <div className="mt-0.5 truncate text-xs text-gray-500">{paramsBrief}</div>
        </div>
      </button>
      {open && (
        <div className="space-y-2 border-t border-gray-100 bg-gray-50 px-10 py-2 text-xs">
          <div>
            <div className="mb-1 font-semibold text-gray-500">Params</div>
            <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded bg-white p-2 text-[11px] text-gray-700 ring-1 ring-gray-200">
              {JSON.stringify(step.params, null, 2)}
            </pre>
          </div>
          {step.error && (
            <div>
              <div className="mb-1 font-semibold text-red-600">Error</div>
              <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded bg-red-50 p-2 text-[11px] text-red-700 ring-1 ring-red-100">
                {step.error}
              </pre>
            </div>
          )}
          {step.result && step.status === "success" && (
            <div>
              <div className="mb-1 font-semibold text-gray-500">Result (truncated)</div>
              <pre className="overflow-x-auto whitespace-pre-wrap break-all rounded bg-white p-2 text-[11px] text-gray-700 ring-1 ring-gray-200">
                {JSON.stringify(step.result, null, 2).slice(0, 500)}
                {JSON.stringify(step.result).length > 500 ? "…" : ""}
              </pre>
            </div>
          )}
        </div>
      )}
    </div>
  );
}

export default function PlanPanel({ plan }: PlanPanelProps) {
  const [open, setOpen] = useState(false);

  const totalMs = plan.steps.reduce((acc, s) => acc + (s.duration_ms ?? 0), 0);
  const successCount = plan.steps.filter((s) => s.status === "success").length;
  const failedCount = plan.steps.filter((s) => s.status === "failed").length;

  const headerColor =
    plan.status === "completed"
      ? "text-emerald-700"
      : plan.status === "partial_failure"
        ? "text-amber-700"
        : plan.status === "failed"
          ? "text-red-700"
          : "text-gray-600";

  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-white shadow-sm">
      <button
        onClick={() => setOpen(!open)}
        className="flex w-full items-center justify-between gap-3 px-3 py-2 text-left hover:bg-gray-50"
      >
        <div className="flex items-center gap-2">
          {open ? (
            <ChevronDown className="h-3.5 w-3.5 text-gray-400" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5 text-gray-400" />
          )}
          <span className="text-xs font-semibold text-gray-700">Execution plan</span>
          <span className={clsx("text-xs font-medium", headerColor)}>{plan.status}</span>
        </div>
        <div className="flex items-center gap-3 text-xs text-gray-500">
          <span>
            {successCount}/{plan.steps.length} steps
          </span>
          {failedCount > 0 && (
            <span className="text-red-600">{failedCount} failed</span>
          )}
          <span className="tabular-nums">{formatDuration(totalMs)}</span>
        </div>
      </button>
      {open && (
        <div>
          {plan.steps.map((step, idx) => (
            <StepRow key={step.id} step={step} index={idx} />
          ))}
        </div>
      )}
    </div>
  );
}
