"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { ArrowLeft, Loader2 } from "lucide-react";
import Button from "@/components/ui/Button";
import StatusBadge from "@/components/ui/StatusBadge";
import { Modal } from "@/components/ui/Modal";
import { Card } from "@/components/ui/Card";
import { useOrder, useUpdateOrderStatus, useCancelOrder } from "@/lib/use-orders";
import { toastSuccess, toastError } from "@/lib/toast";
import { useQueryClient } from "@/lib/api-hooks";

const NEXT_STATUS: Record<string, { label: string; status: string }[]> = {
  pending: [{ label: "Confirm", status: "confirmed" }],
  confirmed: [{ label: "Process", status: "processing" }],
  processing: [{ label: "Ship", status: "shipped" }],
  shipped: [
    { label: "Deliver", status: "delivered" },
    { label: "Return", status: "returned" },
  ],
  delivered: [{ label: "Complete", status: "completed" }],
};

const STATUS_TIMELINE = [
  "pending",
  "confirmed",
  "processing",
  "shipped",
  "delivered",
  "completed",
];

export default function OrderDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading } = useOrder(id);
  const updateStatus = useUpdateOrderStatus();
  const cancelOrder = useCancelOrder();

  const [cancelModalOpen, setCancelModalOpen] = useState(false);
  const [cancelReason, setCancelReason] = useState("");

  const order = data?.data;

  const handleStatusUpdate = (newStatus: string) => {
    updateStatus.mutate(
      { id, status: newStatus },
      {
        onSuccess: () => {
          toastSuccess(`Order status updated to ${newStatus}`);
          queryClient.invalidateQueries({ queryKey: ["orders", id] });
        },
        onError: toastError,
      },
    );
  };

  const handleCancel = () => {
    if (!cancelReason.trim()) return;
    cancelOrder.mutate(
      { id, reason: cancelReason },
      {
        onSuccess: () => {
          toastSuccess("Order cancelled");
          setCancelModalOpen(false);
          setCancelReason("");
          queryClient.invalidateQueries({ queryKey: ["orders", id] });
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

  if (!order) {
    return (
      <div className="text-center py-20 text-gray-500">Order not found</div>
    );
  }

  const canCancel =
    order.status !== "cancelled" &&
    order.status !== "completed" &&
    order.status !== "returned";
  const nextActions = NEXT_STATUS[order.status] ?? [];

  return (
    <div className="space-y-6">
      {}
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.push("/orders")}
          className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-gray-900">Order Details</h1>
          <p className="mt-0.5 font-mono text-sm text-gray-500">{order.id}</p>
        </div>
        <StatusBadge status={order.status} />
      </div>

      {}
      <Card>
        <h2 className="mb-4 text-sm font-semibold text-gray-700">
          Status Timeline
        </h2>
        <div className="flex items-center gap-1">
          {STATUS_TIMELINE.map((s, i) => {
            const idx = STATUS_TIMELINE.indexOf(order.status);
            const isCancelled = order.status === "cancelled" || order.status === "returned";
            const isActive = !isCancelled && STATUS_TIMELINE.indexOf(s) <= idx;
            const isCurrent = s === order.status;
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
                    {s}
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
        {(order.status === "cancelled" || order.status === "returned") && (
          <div className="mt-3">
            <StatusBadge status={order.status} />
            {order.cancel_reason && (
              <span className="ml-2 text-sm text-gray-600">
                Reason: {order.cancel_reason}
              </span>
            )}
          </div>
        )}
      </Card>

      {}
      <div className="grid gap-6 md:grid-cols-3">
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Customer</h2>
          <p className="text-lg font-medium text-gray-900">
            {order.customer_name}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Total</h2>
          <p className="text-lg font-medium text-gray-900">
            ${order.total_amount.toFixed(2)}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Created</h2>
          <p className="text-lg font-medium text-gray-900">
            {new Date(order.created_at).toLocaleString()}
          </p>
        </Card>
      </div>

      {}
      {(nextActions.length > 0 || canCancel) && (
        <div className="flex flex-wrap gap-2">
          {nextActions.map((action) => (
            <Button
              key={action.status}
              onClick={() => handleStatusUpdate(action.status)}
              loading={updateStatus.isPending}
            >
              {action.label}
            </Button>
          ))}
          {canCancel && (
            <Button
              variant="danger"
              onClick={() => setCancelModalOpen(true)}
            >
              Cancel Order
            </Button>
          )}
        </div>
      )}

      {}
      <Card>
        <h2 className="mb-4 text-sm font-semibold text-gray-700">
          Line Items
        </h2>
        {order.items && order.items.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="min-w-full divide-y divide-gray-200">
              <thead className="bg-gray-50">
                <tr>
                  <th className="px-4 py-2 text-left text-xs font-medium uppercase text-gray-500">
                    Product
                  </th>
                  <th className="px-4 py-2 text-right text-xs font-medium uppercase text-gray-500">
                    Qty
                  </th>
                  <th className="px-4 py-2 text-right text-xs font-medium uppercase text-gray-500">
                    Unit Price
                  </th>
                  <th className="px-4 py-2 text-right text-xs font-medium uppercase text-gray-500">
                    Subtotal
                  </th>
                </tr>
              </thead>
              <tbody className="divide-y divide-gray-200">
                {order.items.map((item) => (
                  <tr key={item.id}>
                    <td className="px-4 py-2 text-sm text-gray-900">
                      {item.name}
                    </td>
                    <td className="px-4 py-2 text-right text-sm text-gray-700">
                      {item.quantity}
                    </td>
                    <td className="px-4 py-2 text-right text-sm text-gray-700">
                      ${item.unit_price.toFixed(2)}
                    </td>
                    <td className="px-4 py-2 text-right text-sm font-medium text-gray-900">
                      ${item.subtotal.toFixed(2)}
                    </td>
                  </tr>
                ))}
              </tbody>
              <tfoot>
                <tr className="border-t-2">
                  <td colSpan={3} className="px-4 py-2 text-right text-sm font-semibold text-gray-700">
                    Total
                  </td>
                  <td className="px-4 py-2 text-right text-sm font-bold text-gray-900">
                    ${order.total_amount.toFixed(2)}
                  </td>
                </tr>
              </tfoot>
            </table>
          </div>
        ) : (
          <p className="text-sm text-gray-500">No items</p>
        )}
      </Card>

      {}
      <Modal
        open={cancelModalOpen}
        onClose={() => {
          setCancelModalOpen(false);
          setCancelReason("");
        }}
        title="Cancel Order"
        size="sm"
      >
        <p className="text-sm text-gray-600">
          Are you sure you want to cancel this order? This action cannot be
          undone.
        </p>
        <textarea
          value={cancelReason}
          onChange={(e) => setCancelReason(e.target.value)}
          placeholder="Enter cancellation reason..."
          rows={3}
          className="mt-3 w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
        <div className="mt-4 flex justify-end gap-3">
          <Button
            variant="secondary"
            onClick={() => {
              setCancelModalOpen(false);
              setCancelReason("");
            }}
          >
            Keep Order
          </Button>
          <Button
            variant="danger"
            onClick={handleCancel}
            loading={cancelOrder.isPending}
            disabled={!cancelReason.trim()}
          >
            Cancel Order
          </Button>
        </div>
      </Modal>
    </div>
  );
}
