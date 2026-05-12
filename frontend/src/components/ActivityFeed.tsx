"use client";

import { useEffect, useMemo, useState } from "react";
import { useRealtimeEvents, type RealtimeEvent } from "@/lib/use-realtime-stream";

type FeedTone = "neutral" | "positive" | "warning" | "danger" | "info" | "accent";

const eventMeta: Record<string, { label: string; icon: string; tone: FeedTone }> = {
  "order.created": { label: "Order created", icon: "🧾", tone: "info" },
  "order.updated": { label: "Order updated", icon: "🔄", tone: "neutral" },
  "order.cancelled": { label: "Order cancelled", icon: "✖", tone: "danger" },
  "shipment.created": { label: "Shipment created", icon: "📦", tone: "info" },
  "shipment.updated": { label: "Shipment updated", icon: "🔄", tone: "neutral" },
  "shipment.out_for_delivery": { label: "Out for delivery", icon: "🛵", tone: "accent" },
  "shipment.delivered": { label: "Delivered", icon: "✅", tone: "positive" },
  "shipment.attempted": { label: "Delivery attempted", icon: "⚠", tone: "warning" },
  "shipment.returned": { label: "Returned to sender", icon: "↩", tone: "danger" },
  "shipment.redirected": { label: "Redirected", icon: "➡", tone: "info" },
  "stock.changed": { label: "Stock change", icon: "📊", tone: "neutral" },
  "stock.low": { label: "Low stock alert", icon: "🔻", tone: "warning" },
  "notification.new": { label: "Notification", icon: "🔔", tone: "info" },
  "analytics.updated": { label: "Analytics update", icon: "📈", tone: "neutral" },
};

const toneClass: Record<FeedTone, string> = {
  neutral: "border-gray-200 bg-white",
  positive: "border-green-200 bg-green-50",
  warning: "border-amber-200 bg-amber-50",
  danger: "border-red-200 bg-red-50",
  info: "border-blue-200 bg-blue-50",
  accent: "border-purple-200 bg-purple-50",
};

function relative(ts: string): string {
  const t = new Date(ts).getTime();
  const diff = Math.max(0, Date.now() - t);
  if (diff < 5_000) return "just now";
  if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
  return new Date(ts).toLocaleTimeString();
}

function summarize(ev: RealtimeEvent): string {
  const d = ev.data || {};
  if (typeof d === "object" && d) {
    const obj = d as Record<string, unknown>;
    if (typeof obj.order_id === "string") return `order ${(obj.order_id as string).slice(0, 8)}…`;
    if (typeof obj.shipment_id === "string") return `shipment ${(obj.shipment_id as string).slice(0, 8)}…`;
    if (typeof obj.tracking_number === "string") return obj.tracking_number as string;
    if (typeof obj.notification_id === "string") return `to user ${(obj.user_id as string)?.slice(0, 8) || "?"}`;
    if (typeof obj.product_id === "string") return `product ${(obj.product_id as string).slice(0, 8)}…`;
    if (typeof obj.metric === "string") return `${obj.metric}`;
  }
  return ev.subject || ev.type;
}

export default function ActivityFeed({ onClose }: { onClose: () => void }) {
  const events = useRealtimeEvents(80);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    const id = setInterval(() => setTick((t) => t + 1), 5000);
    return () => clearInterval(id);
  }, []);

  const items = useMemo(() => events.map((e, i) => ({ ev: e, key: `${e.timestamp}-${i}` })), [events]);

  return (
    <aside className="flex h-full w-96 flex-col border-l border-gray-200 bg-gray-50">
      <header className="flex items-center justify-between border-b border-gray-200 bg-white px-4 py-3">
        <div>
          <h2 className="text-sm font-semibold text-gray-900">Activity feed</h2>
          <p className="text-xs text-gray-500">Live operations stream</p>
        </div>
        <button
          type="button"
          className="rounded p-1 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
          onClick={onClose}
          aria-label="Close activity feed"
        >
          ✕
        </button>
      </header>
      <div className="flex-1 space-y-2 overflow-y-auto p-3" data-tick={tick}>
        {items.length === 0 ? (
          <div className="flex h-full items-center justify-center text-center text-xs text-gray-400">
            Live events will appear here.
            <br />Enable the simulator on the admin page to see traffic.
          </div>
        ) : (
          items.map(({ ev, key }) => {
            const meta = eventMeta[ev.type] || { label: ev.type, icon: "•", tone: "neutral" as FeedTone };
            return (
              <div
                key={key}
                className={`rounded-md border px-3 py-2 text-xs shadow-sm transition-all ${toneClass[meta.tone]}`}
              >
                <div className="flex items-center justify-between">
                  <span className="font-medium text-gray-900">
                    <span className="mr-1.5">{meta.icon}</span>
                    {meta.label}
                  </span>
                  <span className="text-[10px] text-gray-500">{relative(ev.timestamp)}</span>
                </div>
                <div className="mt-1 truncate text-[11px] text-gray-600">{summarize(ev)}</div>
              </div>
            );
          })
        )}
      </div>
    </aside>
  );
}
