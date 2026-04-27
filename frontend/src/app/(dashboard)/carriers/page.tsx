"use client";

import { useState, useCallback } from "react";
import { Plus } from "lucide-react";
import { useForm } from "react-hook-form";
import DataTable, { type Column } from "@/components/ui/DataTable";
import StatusBadge from "@/components/ui/StatusBadge";
import Button from "@/components/ui/Button";
import { Modal } from "@/components/ui/Modal";
import { FormField, Input, Select } from "@/components/ui/FormField";
import { useCarriers, useCreateCarrier, useUpdateCarrier, type Carrier, type CreateCarrierInput, type UpdateCarrierInput } from "@/lib/use-carriers";
import { toastSuccess, toastError } from "@/lib/toast";
import { useQueryClient } from "@/lib/api-hooks";

const CARRIER_TYPES = [
  { value: "", label: "All types" },
  { value: "ground", label: "Ground" },
  { value: "air", label: "Air" },
  { value: "sea", label: "Sea" },
];

const CARRIER_TYPE_OPTIONS = [
  { value: "ground", label: "Ground" },
  { value: "air", label: "Air" },
  { value: "sea", label: "Sea" },
];

const columns: Column<Carrier>[] = [
  { key: "name", header: "Name", sortable: true },
  {
    key: "type",
    header: "Type",
    sortable: true,
    render: (row) => (
      <span className="inline-flex items-center rounded-full bg-gray-100 px-2.5 py-0.5 text-xs font-medium text-gray-800 capitalize">
        {row.type}
      </span>
    ),
  },
  {
    key: "cost_per_km",
    header: "Cost / km",
    sortable: true,
    render: (row) => `$${row.cost_per_km.toFixed(2)}`,
    className: "text-right",
  },
  {
    key: "is_active",
    header: "Status",
    render: (row) => (
      <StatusBadge status={row.is_active ? "active" : "inactive"} />
    ),
  },
  {
    key: "created_at",
    header: "Created",
    sortable: true,
    render: (row) => new Date(row.created_at).toLocaleDateString(),
  },
];

export default function CarriersPage() {
  const queryClient = useQueryClient();
  const [page, setPage] = useState(1);
  const [sortField, setSortField] = useState("created_at");
  const [sortDesc, setSortDesc] = useState(true);
  const [typeFilter, setTypeFilter] = useState("");
  const [createModalOpen, setCreateModalOpen] = useState(false);
  const [editCarrier, setEditCarrier] = useState<Carrier | null>(null);

  const { data, isLoading } = useCarriers({
    page,
    pageSize: 10,
    sortField,
    sortDesc,
    type: typeFilter || undefined,
  });

  const createMutation = useCreateCarrier();
  const updateMutation = useUpdateCarrier();

  const carriers = data?.data ?? [];
  const total = data?.meta?.total ?? 0;

  const handleSort = useCallback((field: string, desc: boolean) => {
    setSortField(field);
    setSortDesc(desc);
    setPage(1);
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Carriers</h1>
        <Button onClick={() => setCreateModalOpen(true)}>
          <Plus className="h-4 w-4" />
          New Carrier
        </Button>
      </div>

      {}
      <div className="flex flex-wrap items-center gap-3">
        <select
          value={typeFilter}
          onChange={(e) => {
            setTypeFilter(e.target.value);
            setPage(1);
          }}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          {CARRIER_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>

      {}
      <DataTable<Carrier>
        columns={columns}
        data={carriers}
        total={total}
        page={page}
        pageSize={10}
        onPageChange={setPage}
        sortField={sortField}
        sortDesc={sortDesc}
        onSort={handleSort}
        loading={isLoading}
        emptyMessage="No carriers found"
        rowKey={(row) => row.id}
        onRowClick={(row) => setEditCarrier(row)}
      />

      {}
      <CreateCarrierModal
        open={createModalOpen}
        onClose={() => setCreateModalOpen(false)}
        onSubmit={(values) => {
          createMutation.mutate(values, {
            onSuccess: () => {
              toastSuccess("Carrier created");
              setCreateModalOpen(false);
            },
            onError: toastError,
          });
        }}
        loading={createMutation.isPending}
      />

      {}
      {editCarrier && (
        <EditCarrierModal
          carrier={editCarrier}
          open={true}
          onClose={() => setEditCarrier(null)}
          onSubmit={(values) => {
            updateMutation.mutate(
              { id: editCarrier.id, ...values },
              {
                onSuccess: () => {
                  toastSuccess("Carrier updated");
                  setEditCarrier(null);
                  queryClient.invalidateQueries({ queryKey: ["carriers"] });
                },
                onError: toastError,
              },
            );
          }}
          loading={updateMutation.isPending}
        />
      )}
    </div>
  );
}

function CreateCarrierModal({
  open,
  onClose,
  onSubmit,
  loading,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: CreateCarrierInput) => void;
  loading: boolean;
}) {
  const { register, handleSubmit, reset, formState: { errors } } = useForm<CreateCarrierInput>();

  const handleClose = () => {
    reset();
    onClose();
  };

  return (
    <Modal open={open} onClose={handleClose} title="New Carrier" size="sm">
      <form
        onSubmit={handleSubmit((values) => {
          onSubmit({ ...values, cost_per_km: Number(values.cost_per_km) });
        })}
        className="space-y-4"
      >
        <FormField label="Name" required error={errors.name}>
          <Input
            registration={register("name", { required: "Name is required" })}
            error={errors.name}
          />
        </FormField>
        <FormField label="Type" required error={errors.type}>
          <Select
            registration={register("type", { required: "Type is required" })}
            options={CARRIER_TYPE_OPTIONS}
            placeholder="Select type"
            error={errors.type}
          />
        </FormField>
        <FormField label="Cost per km ($)" required error={errors.cost_per_km}>
          <Input
            type="number"
            step="0.01"
            registration={register("cost_per_km", {
              required: "Cost is required",
              min: { value: 0.01, message: "Must be > 0" },
            })}
            error={errors.cost_per_km}
          />
        </FormField>
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" loading={loading}>
            Create
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function EditCarrierModal({
  carrier,
  open,
  onClose,
  onSubmit,
  loading,
}: {
  carrier: Carrier;
  open: boolean;
  onClose: () => void;
  onSubmit: (values: UpdateCarrierInput) => void;
  loading: boolean;
}) {
  const { register, handleSubmit, formState: { errors } } = useForm<UpdateCarrierInput>({
    defaultValues: {
      name: carrier.name,
      type: carrier.type,
      cost_per_km: carrier.cost_per_km,
      is_active: carrier.is_active,
    },
  });

  return (
    <Modal open={open} onClose={onClose} title="Edit Carrier" size="sm">
      <form
        onSubmit={handleSubmit((values) => {
          onSubmit({
            ...values,
            cost_per_km: Number(values.cost_per_km),
            is_active: values.is_active === true || values.is_active === ("true" as unknown as boolean),
          });
        })}
        className="space-y-4"
      >
        <FormField label="Name" required error={errors.name}>
          <Input
            registration={register("name", { required: "Name is required" })}
            error={errors.name}
          />
        </FormField>
        <FormField label="Type" required error={errors.type}>
          <Select
            registration={register("type", { required: "Type is required" })}
            options={CARRIER_TYPE_OPTIONS}
            error={errors.type}
          />
        </FormField>
        <FormField label="Cost per km ($)" required error={errors.cost_per_km}>
          <Input
            type="number"
            step="0.01"
            registration={register("cost_per_km", {
              required: "Cost is required",
              min: { value: 0.01, message: "Must be > 0" },
            })}
            error={errors.cost_per_km}
          />
        </FormField>
        <FormField label="Status">
          <Select
            registration={register("is_active")}
            options={[
              { value: "true", label: "Active" },
              { value: "false", label: "Inactive" },
            ]}
          />
        </FormField>
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={onClose}>
            Cancel
          </Button>
          <Button type="submit" loading={loading}>
            Save
          </Button>
        </div>
      </form>
    </Modal>
  );
}
