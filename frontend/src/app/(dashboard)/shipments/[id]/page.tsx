"use client";

import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Loader2 } from "lucide-react";
import Button from "@/components/ui/Button";
import StatusBadge from "@/components/ui/StatusBadge";
import { Card } from "@/components/ui/Card";
import { useShipment, useUpdateShipmentStatus } from "@/lib/use-shipments";
import { toastSuccess, toastError } from "@/lib/toast";
import { useQueryClient } from "@/lib/api-hooks";

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
  const updateStatus = useUpdateShipmentStatus();

  const shipment = data?.data;

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

      <div className="grid gap-6 md:grid-cols-2">
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Address</h2>
          <p className="text-gray-900">{shipment.address}</p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Created</h2>
          <p className="text-gray-900">
            {new Date(shipment.created_at).toLocaleString()}
          </p>
        </Card>
      </div>

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
