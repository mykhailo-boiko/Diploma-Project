"use client";

import { useRealtimeStream } from "@/lib/use-realtime-stream";

export default function LiveIndicator({ onToggleFeed }: { onToggleFeed?: () => void }) {
  const { status } = useRealtimeStream();

  const dotClass =
    status === "open"
      ? "bg-emerald-500 shadow-[0_0_0_3px_rgba(16,185,129,0.18)] animate-pulse"
      : status === "connecting"
      ? "bg-amber-500"
      : status === "error"
      ? "bg-red-500"
      : "bg-gray-400";

  const label =
    status === "open"
      ? "Live"
      : status === "connecting"
      ? "Connecting…"
      : status === "error"
      ? "Reconnecting…"
      : "Offline";

  const button = (
    <span className="inline-flex items-center gap-1.5 text-xs font-medium text-gray-700">
      <span className={`inline-block h-2 w-2 rounded-full ${dotClass}`} />
      {label}
    </span>
  );

  if (!onToggleFeed) return button;

  return (
    <button
      type="button"
      onClick={onToggleFeed}
      className="rounded-md border border-gray-200 bg-white px-2.5 py-1 hover:border-gray-300 hover:bg-gray-50"
      title="Toggle activity feed"
    >
      {button}
    </button>
  );
}
