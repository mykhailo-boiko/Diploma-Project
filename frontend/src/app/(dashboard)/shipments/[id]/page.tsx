"use client";

import { useParams, useRouter } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Copy, ExternalLink, Loader2, Package } from "lucide-react";
import Button from "@/components/ui/Button";
import StatusBadge from "@/components/ui/StatusBadge";
import { Card } from "@/components/ui/Card";
import {
  useShipment,
  useShipmentTimeline,
  useUpdateShipmentStatus,
} from "@/lib/use-shipments";
import { toastSuccess, toastError } from "@/lib/toast";
import { useQueryClient } from "@/lib/api-hooks";
import TrackingTimeline from "@/components/TrackingTimeline";

const NEXT_STATUS: Record<string, { label: string; status: string }[]> = {
  created: [{ label: "Pick Up", status: "picked_up" }],
  picked_up: [{ label: "In Transit", status: "in_transit" }],
  in_transit: [
    { label: "Deliver", status: "delivered" },
    { label: "Fail", status: "failed" },
  ],
  delivered: [{ label: "Return", status: "returned" }],
  failed: [{ label: "Return", status: "returned" }],
};

const STATUS_TIMELINE = [
  "created",
  "picked_up",
  "in_transit",
  "delivered",
];

export default function ShipmentDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading } = useShipment(id);
  const { data: timelineData } = useShipmentTimeline(id);
  const updateStatus = useUpdateShipmentStatus();

  const shipment = data?.data;
  const timeline = timelineData?.data;

  const handleStatusUpdate = (newStatus: string) => {
    updateStatus.mutate(
      { id, status: newStatus },
      {
        onSuccess: () => {
          toastSuccess(`Shipment status updated to ${newStatus.replace("_", " ")}`);
          queryClient.invalidateQueries({ queryKey: ["shipments", id] });
        },
        onError: toastError,
      },
    );
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!shipment) {
    return (
      <div className="text-center py-20 text-gray-500">Shipment not found</div>
    );
  }

  const nextActions = NEXT_STATUS[shipment.status] ?? [];

  return (
    <div className="space-y-6">
      {}
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.push("/shipments")}
          className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-gray-900">Shipment Details</h1>
          <p className="mt-0.5 font-mono text-sm text-gray-500">{shipment.id}</p>
        </div>
        <StatusBadge status={shipment.status} />
      </div>

      {}
      <Card>
        <h2 className="mb-4 text-sm font-semibold text-gray-700">
          Status Timeline
        </h2>
        <div className="flex items-center gap-1">
          {STATUS_TIMELINE.map((s, i) => {
            const idx = STATUS_TIMELINE.indexOf(shipment.status);
            const isFailed = shipment.status === "failed" || shipment.status === "returned";
            const isActive = !isFailed && STATUS_TIMELINE.indexOf(s) <= idx;
            const isCurrent = s === shipment.status;
            return (
              <div key={s} className="flex flex-1 items-center">
                <div className="flex flex-col items-center flex-1">
                  <div
                    className={`h-3 w-3 rounded-full border-2 ${
                      isCurrent
                        ? "border-blue-500 bg-blue-500"
                        : isActive
                          ? "border-green-500 bg-green-500"
                          : "border-gray-300 bg-white"
                    }`}
                  />
                  <span
                    className={`mt-1 text-[10px] capitalize ${
                      isCurrent ? "font-semibold text-blue-600" : "text-gray-500"
                    }`}
                  >
                    {s.replace("_", " ")}
                  </span>
                </div>
                {i < STATUS_TIMELINE.length - 1 && (
                  <div
                    className={`h-0.5 flex-1 ${
                      isActive && STATUS_TIMELINE.indexOf(s) < idx
                        ? "bg-green-400"
                        : "bg-gray-200"
                    }`}
                  />
                )}
              </div>
            );
          })}
        </div>
        {(shipment.status === "failed" || shipment.status === "returned") && (
          <div className="mt-3">
            <StatusBadge status={shipment.status} />
          </div>
        )}
      </Card>

      {}
      <div className="grid gap-6 md:grid-cols-3">
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Order ID</h2>
          <p className="font-mono text-sm text-gray-900">{shipment.order_id}</p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Carrier ID</h2>
          <p className="font-mono text-sm text-gray-900">{shipment.carrier_id}</p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Warehouse ID</h2>
          <p className="font-mono text-sm text-gray-900">{shipment.warehouse_id}</p>
        </Card>
      </div>

      {}
      {shipment.tracking_number && (
        <Card>
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-gray-500">
                <Package className="h-3.5 w-3.5" />
                Tracking number
              </div>
              <div className="mt-1 font-mono text-lg font-semibold text-gray-900">
                {shipment.tracking_number}
              </div>
            </div>
            <div className="flex items-center gap-2">
              <button
                onClick={() => {
                  navigator.clipboard.writeText(shipment.tracking_number);
                  toastSuccess("Tracking number copied");
                }}
                className="inline-flex items-center gap-1 rounded-md border border-gray-200 px-3 py-1.5 text-xs font-medium text-gray-700 hover:bg-gray-50"
              >
                <Copy className="h-3.5 w-3.5" /> Copy
              </button>
              <Link
                href={`/track/${shipment.tracking_number}`}
                target="_blank"
                className="inline-flex items-center gap-1 rounded-md bg-blue-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-blue-700"
              >
                <ExternalLink className="h-3.5 w-3.5" /> Open public tracking
              </Link>
            </div>
          </div>
          {(shipment.estimated_delivery_at ||
            shipment.delivered_at ||
            shipment.current_location_city ||
            shipment.current_location_hub) && (
            <div className="mt-4 grid gap-4 sm:grid-cols-3">
              {shipment.estimated_delivery_at && (
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">ETA</div>
                  <div className="mt-1 text-sm text-gray-900">
                    {new Date(shipment.estimated_delivery_at).toLocaleString()}
                  </div>
                </div>
              )}
              {shipment.delivered_at && (
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">
                    Delivered at
                  </div>
                  <div className="mt-1 text-sm text-gray-900">
                    {new Date(shipment.delivered_at).toLocaleString()}
                  </div>
                </div>
              )}
              {(shipment.current_location_city || shipment.current_location_hub) && (
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">
                    Current location
                  </div>
                  <div className="mt-1 text-sm text-gray-900">
                    {shipment.current_location_hub ?? shipment.current_location_city}
                  </div>
                </div>
              )}
              {shipment.delivery_attempts > 0 && (
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">
                    Delivery attempts
                  </div>
                  <div className="mt-1 text-sm text-gray-900">
                    {shipment.delivery_attempts}
                  </div>
                </div>
              )}
              {shipment.delivery_signature && (
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">
                    Signed by
                  </div>
                  <div className="mt-1 text-sm text-gray-900">
                    {shipment.delivery_signature}
                  </div>
                </div>
              )}
            </div>
          )}
        </Card>
      )}

      {}
      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <h2 className="mb-3 text-sm font-semibold text-gray-700">Recipient</h2>
          {shipment.recipient && Object.keys(shipment.recipient).length > 0 ? (
            <dl className="space-y-1 text-sm text-gray-700">
              {shipment.recipient.full_name && (
                <div>
                  <dt className="inline text-gray-500">Name: </dt>
                  <dd className="inline text-gray-900">{shipment.recipient.full_name}</dd>
                </div>
              )}
              {shipment.recipient.phone && (
                <div>
                  <dt className="inline text-gray-500">Phone: </dt>
                  <dd className="inline font-mono text-gray-900">
                    {shipment.recipient.phone}
                  </dd>
                </div>
              )}
              {shipment.recipient.email && (
                <div>
                  <dt className="inline text-gray-500">Email: </dt>
                  <dd className="inline text-gray-900">{shipment.recipient.email}</dd>
                </div>
              )}
              {(shipment.recipient.street || shipment.recipient.city) && (
                <div>
                  <dt className="inline text-gray-500">Address: </dt>
                  <dd className="inline text-gray-900">
                    {[
                      shipment.recipient.street,
                      shipment.recipient.city,
                      shipment.recipient.postcode,
                      shipment.recipient.country,
                    ]
                      .filter(Boolean)
                      .join(", ")}
                  </dd>
                </div>
              )}
            </dl>
          ) : (
            <p className="text-sm text-gray-500">No recipient details</p>
          )}
        </Card>
        <Card>
          <h2 className="mb-3 text-sm font-semibold text-gray-700">Sender</h2>
          {shipment.sender && Object.keys(shipment.sender).length > 0 ? (
            <dl className="space-y-1 text-sm text-gray-700">
              {shipment.sender.company && (
                <div>
                  <dt className="inline text-gray-500">Company: </dt>
                  <dd className="inline text-gray-900">{shipment.sender.company}</dd>
                </div>
              )}
              {shipment.sender.phone && (
                <div>
                  <dt className="inline text-gray-500">Phone: </dt>
                  <dd className="inline font-mono text-gray-900">{shipment.sender.phone}</dd>
                </div>
              )}
              {shipment.sender.email && (
                <div>
                  <dt className="inline text-gray-500">Email: </dt>
                  <dd className="inline text-gray-900">{shipment.sender.email}</dd>
                </div>
              )}
              {(shipment.sender.street || shipment.sender.city) && (
                <div>
                  <dt className="inline text-gray-500">From: </dt>
                  <dd className="inline text-gray-900">
                    {[
                      shipment.sender.street,
                      shipment.sender.city,
                      shipment.sender.country,
                    ]
                      .filter(Boolean)
                      .join(", ")}
                  </dd>
                </div>
              )}
            </dl>
          ) : (
            <p className="text-sm text-gray-500">No sender details</p>
          )}
        </Card>
      </div>

      {}
      <Card>
        <h2 className="mb-4 text-sm font-semibold text-gray-700">Tracking Timeline</h2>
        {timeline ? (
          <TrackingTimeline events={timeline.events} />
        ) : (
          <div className="text-sm text-gray-500">Loading timeline…</div>
        )}
      </Card>

      {timeline && timeline.delivery_attempts.length > 0 && (
        <Card>
          <h2 className="mb-3 text-sm font-semibold text-gray-700">Delivery attempts</h2>
          <ul className="space-y-2 text-sm">
            {timeline.delivery_attempts.map((a) => (
              <li
                key={a.id}
                className="flex flex-wrap items-baseline justify-between gap-3 rounded-md border border-gray-100 bg-gray-50 px-3 py-2"
              >
                <div>
                  <span className="font-medium text-gray-900">
                    Attempt #{a.attempt_number}
                  </span>
                  <span className="ml-2 text-gray-700">— {a.reason.replace(/_/g, " ")}</span>
                  {a.notes && <p className="mt-0.5 text-xs text-gray-500">{a.notes}</p>}
                </div>
                <div className="text-xs tabular-nums text-gray-500">
                  {new Date(a.occurred_at).toLocaleString()}
                </div>
              </li>
            ))}
          </ul>
        </Card>
      )}

      {}
      {nextActions.length > 0 && (
        <div className="flex flex-wrap gap-2">
          {nextActions.map((action) => (
            <Button
              key={action.status}
              variant={action.status === "failed" ? "danger" : "primary"}
              onClick={() => handleStatusUpdate(action.status)}
              loading={updateStatus.isPending}
            >
              {action.label}
            </Button>
          ))}
        </div>
      )}
    </div>
  );
}
