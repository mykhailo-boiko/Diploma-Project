"use client";

import { useState } from "react";
import { useParams, useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { ArrowLeft, Loader2, Pencil } from "lucide-react";
import Button from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import StatusBadge from "@/components/ui/StatusBadge";
import { Modal } from "@/components/ui/Modal";
import { FormField, Input, Textarea } from "@/components/ui/FormField";
import DataTable, { type Column } from "@/components/ui/DataTable";
import { useWarehouse, useUpdateWarehouse } from "@/lib/use-warehouses";
import { useStock, type Stock } from "@/lib/use-stock";
import { toastSuccess, toastError } from "@/lib/toast";
import { useQueryClient } from "@/lib/api-hooks";
import { safeLocale } from "@/lib/format";

interface EditFormValues {
  name: string;
  address: string;
  is_active: string;
}

export default function WarehouseDetailPage() {
  const { id } = useParams<{ id: string }>();
  const router = useRouter();
  const queryClient = useQueryClient();

  const { data, isLoading } = useWarehouse(id);
  const updateWarehouse = useUpdateWarehouse();
  const stockQuery = useStock({
    page: 1,
    pageSize: 50,
    warehouseId: id,
  });

  const [editOpen, setEditOpen] = useState(false);
  const warehouse = data?.data;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<EditFormValues>();

  const openEdit = () => {
    if (!warehouse) return;
    reset({
      name: warehouse.name,
      address: warehouse.address || "",
      is_active: warehouse.is_active ? "true" : "false",
    });
    setEditOpen(true);
  };

  const onSubmit = (data: EditFormValues) => {
    updateWarehouse.mutate(
      {
        id,
        name: data.name,
        address: data.address || undefined,
        is_active: data.is_active === "true",
      },
      {
        onSuccess: () => {
          toastSuccess("Warehouse updated");
          setEditOpen(false);
          queryClient.invalidateQueries({ queryKey: ["warehouses", id] });
        },
        onError: toastError,
      },
    );
  };

  const stocks = stockQuery.data?.data ?? [];
  const totalQty = stocks.reduce((s, r) => s + r.quantity, 0);
  const totalReserved = stocks.reduce((s, r) => s + r.reserved, 0);
  const totalAvailable = stocks.reduce((s, r) => s + r.available, 0);

  const stockColumns: Column<Stock>[] = [
    {
      key: "product_id",
      header: "Product ID",
      render: (row) => (
        <span className="font-mono text-xs text-gray-600">
          {row.product_id.slice(0, 8)}...
        </span>
      ),
    },
    { key: "quantity", header: "Quantity", className: "text-right" },
    {
      key: "reserved",
      header: "Reserved",
      className: "text-right",
      render: (row) => (
        <span className={row.reserved > 0 ? "text-amber-600 font-medium" : ""}>
          {row.reserved}
        </span>
      ),
    },
    {
      key: "available",
      header: "Available",
      className: "text-right",
      render: (row) => (
        <span
          className={`font-medium ${
            row.min_threshold > 0 && row.available < row.min_threshold
              ? "text-red-600"
              : "text-green-600"
          }`}
        >
          {row.available}
        </span>
      ),
    },
    {
      key: "min_threshold",
      header: "Threshold",
      className: "text-right",
      render: (row) =>
        row.min_threshold > 0 ? (
          row.min_threshold
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
  ];

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <Loader2 className="h-8 w-8 animate-spin text-gray-400" />
      </div>
    );
  }

  if (!warehouse) {
    return (
      <div className="text-center py-20 text-gray-500">
        Warehouse not found
      </div>
    );
  }

  const capacityMax = Math.max(totalQty, 1000);
  const capacityPct = Math.min((totalQty / capacityMax) * 100, 100);

  return (
    <div className="space-y-6">
      {}
      <div className="flex items-center gap-4">
        <button
          onClick={() => router.push("/warehouses")}
          className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
        >
          <ArrowLeft className="h-5 w-5" />
        </button>
        <div className="flex-1">
          <h1 className="text-2xl font-bold text-gray-900">{warehouse.name}</h1>
          <p className="mt-0.5 text-sm text-gray-500">
            {warehouse.address || "No address"}
          </p>
        </div>
        <StatusBadge
          status={warehouse.is_active ? "active" : "inactive"}
        />
        <Button variant="secondary" onClick={openEdit}>
          <Pencil className="h-4 w-4" />
          Edit
        </Button>
      </div>

      {}
      <div className="grid gap-6 md:grid-cols-4">
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">
            Total Quantity
          </h2>
          <p className="text-2xl font-bold text-gray-900">
            {safeLocale(totalQty)}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">Reserved</h2>
          <p className="text-2xl font-bold text-amber-600">
            {safeLocale(totalReserved)}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">
            Available
          </h2>
          <p className="text-2xl font-bold text-green-600">
            {safeLocale(totalAvailable)}
          </p>
        </Card>
        <Card>
          <h2 className="mb-2 text-sm font-semibold text-gray-700">
            Products Stocked
          </h2>
          <p className="text-2xl font-bold text-gray-900">{stocks.length}</p>
        </Card>
      </div>

      {}
      <Card>
        <h2 className="mb-3 text-sm font-semibold text-gray-700">
          Warehouse Load
        </h2>
        <div className="h-4 w-full overflow-hidden rounded-full bg-gray-200">
          <div
            className={`h-full rounded-full transition-all ${
              capacityPct > 80
                ? "bg-red-500"
                : capacityPct > 50
                  ? "bg-yellow-500"
                  : "bg-green-500"
            }`}
            style={{ width: `${capacityPct}%` }}
          />
        </div>
        <p className="mt-1 text-right text-xs text-gray-500">
          {safeLocale(totalQty)} units
        </p>
      </Card>

      {}
      <Card padding={false}>
        <div className="p-4 pb-0">
          <h2 className="text-sm font-semibold text-gray-700">
            Stock in Warehouse
          </h2>
        </div>
        <div className="p-4">
          <DataTable<Stock>
            columns={stockColumns}
            data={stocks}
            total={stocks.length}
            page={1}
            pageSize={50}
            onPageChange={() => {}}
            loading={stockQuery.isLoading}
            emptyMessage="No stock entries for this warehouse"
            rowKey={(row) => row.id}
          />
        </div>
      </Card>

      {}
      <Modal
        open={editOpen}
        onClose={() => setEditOpen(false)}
        title="Edit Warehouse"
        size="sm"
      >
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <FormField label="Name" required error={errors.name}>
            <Input
              registration={register("name", { required: "Name is required" })}
              error={errors.name}
            />
          </FormField>
          <FormField label="Address">
            <Textarea registration={register("address")} rows={2} />
          </FormField>
          <FormField label="Status">
            <select
              {...register("is_active")}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              <option value="true">Active</option>
              <option value="false">Inactive</option>
            </select>
          </FormField>
          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => setEditOpen(false)}
            >
              Cancel
            </Button>
            <Button type="submit" loading={updateWarehouse.isPending}>
              Save Changes
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
