"use client";

import clsx from "clsx";
import {
  Package,
  CheckCircle2,
  Truck,
  Building2,
  MapPin,
  AlertTriangle,
  RotateCcw,
  Clock,
  Home,
  RefreshCw,
  XCircle,
} from "lucide-react";

export interface ShipmentEvent {
  id: string;
  event_type: string;
  location_city?: string;
  location_hub?: string;
  notes?: string;
  occurred_at: string;
  recorded_by: string;
}

interface TrackingTimelineProps {
  events: ShipmentEvent[];
  compact?: boolean;
}

function iconFor(eventType: string) {
  switch (eventType) {
    case "label_created":
      return Package;
    case "awaiting_pickup":
      return Clock;
    case "picked_up":
      return Truck;
    case "in_transit":
      return Truck;
    case "at_hub":
    case "hub_arrived":
    case "hub_departed":
      return Building2;
    case "out_for_delivery":
      return MapPin;
    case "delivery_attempted":
      return AlertTriangle;
    case "held_at_office":
      return Home;
    case "delivered":
      return CheckCircle2;
    case "returned_to_sender":
    case "returned":
      return RotateCcw;
    case "redirected":
      return RefreshCw;
    case "cancelled":
      return XCircle;
    case "recipient_updated":
    case "sender_updated":
      return RefreshCw;
    default:
      return Clock;
  }
}

function tintFor(eventType: string): { bg: string; ring: string; icon: string } {
  switch (eventType) {
    case "delivered":
      return {
        bg: "bg-emerald-50",
        ring: "ring-emerald-200",
        icon: "text-emerald-600",
      };
    case "delivery_attempted":
    case "returned_to_sender":
    case "returned":
    case "cancelled":
      return {
        bg: "bg-red-50",
        ring: "ring-red-200",
        icon: "text-red-600",
      };
    case "out_for_delivery":
    case "held_at_office":
      return {
        bg: "bg-amber-50",
        ring: "ring-amber-200",
        icon: "text-amber-600",
      };
    case "redirected":
    case "recipient_updated":
    case "sender_updated":
      return {
        bg: "bg-violet-50",
        ring: "ring-violet-200",
        icon: "text-violet-600",
      };
    default:
      return {
        bg: "bg-blue-50",
        ring: "ring-blue-200",
        icon: "text-blue-600",
      };
  }
}

function formatTimestamp(ts: string): string {
  try {
    const d = new Date(ts);
    return d.toLocaleString("en-GB", {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return ts;
  }
}

function humanize(eventType: string): string {
  return eventType.replace(/_/g, " ").replace(/\b\w/g, (c) => c.toUpperCase());
}

export default function TrackingTimeline({ events, compact = false }: TrackingTimelineProps) {
  if (events.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-gray-300 p-6 text-center text-sm text-gray-500">
        No tracking events yet.
      </div>
    );
  }

  const sorted = [...events].sort(
    (a, b) => new Date(b.occurred_at).getTime() - new Date(a.occurred_at).getTime(),
  );

  return (
    <ol className="relative space-y-4">
      <span
        className="absolute left-5 top-2 bottom-2 w-px bg-gradient-to-b from-gray-200 via-gray-200 to-transparent"
        aria-hidden="true"
      />
      {sorted.map((event, idx) => {
        const Icon = iconFor(event.event_type);
        const tint = tintFor(event.event_type);
        const isLatest = idx === 0;
        return (
          <li key={event.id} className="relative flex gap-4 pl-0">
            <div
              className={clsx(
                "z-10 flex h-10 w-10 flex-shrink-0 items-center justify-center rounded-full ring-2",
                tint.bg,
                tint.ring,
                isLatest && "ring-4 ring-offset-1 ring-offset-white",
              )}
            >
              <Icon className={clsx("h-5 w-5", tint.icon)} />
            </div>
            <div className={clsx("min-w-0 flex-1 pb-2", compact && "text-sm")}>
              <div className="flex flex-wrap items-baseline gap-x-3">
                <span className="font-medium text-gray-900">{humanize(event.event_type)}</span>
                {isLatest && (
                  <span className="rounded-full bg-blue-50 px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide text-blue-700 ring-1 ring-blue-100">
                    latest
                  </span>
                )}
              </div>
              {(event.location_city || event.location_hub) && (
                <div className="mt-0.5 flex items-center gap-1 text-xs text-gray-600">
                  <MapPin className="h-3 w-3" />
                  <span>
                    {[event.location_hub, event.location_city]
                      .filter(Boolean)
                      .join(" · ")}
                  </span>
                </div>
              )}
              {event.notes && (
                <p className="mt-1 text-xs text-gray-500">{event.notes}</p>
              )}
              <div className="mt-1 text-[11px] tabular-nums text-gray-400">
                {formatTimestamp(event.occurred_at)} · {event.recorded_by}
              </div>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
