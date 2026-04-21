"use client";

import clsx from "clsx";

type StatusVariant =
  | "success"
  | "warning"
  | "error"
  | "info"
  | "neutral"
  | "primary";

const variantStyles: Record<StatusVariant, string> = {
  success: "bg-green-50 text-green-700 ring-green-600/20",
  warning: "bg-yellow-50 text-yellow-700 ring-yellow-600/20",
  error: "bg-red-50 text-red-700 ring-red-600/20",
  info: "bg-blue-50 text-blue-700 ring-blue-600/20",
  neutral: "bg-gray-50 text-gray-600 ring-gray-500/20",
  primary: "bg-indigo-50 text-indigo-700 ring-indigo-600/20",
};

const statusColorMap: Record<string, StatusVariant> = {
  pending: "warning",
  confirmed: "info",
  processing: "primary",
  shipped: "info",
  delivered: "success",
  completed: "success",
  cancelled: "error",
  returned: "error",
  created: "neutral",
  picked_up: "info",
  in_transit: "info",
  failed: "error",
  active: "success",
  inactive: "neutral",
  read: "neutral",
};

interface StatusBadgeProps {
  status: string;
  variant?: StatusVariant;
  className?: string;
}

export default function StatusBadge({
  status,
  variant,
  className,
}: StatusBadgeProps) {
  const resolvedVariant = variant || statusColorMap[status] || "neutral";
  const label = status.replace(/_/g, " ");

  return (
    <span
      className={clsx(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium capitalize ring-1 ring-inset",
        variantStyles[resolvedVariant],
        className,
      )}
    >
      {label}
    </span>
  );
}
