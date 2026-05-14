"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Plus, Search, X } from "lucide-react";
import DataTable, { type Column } from "@/components/ui/DataTable";
import StatusBadge from "@/components/ui/StatusBadge";
import Button from "@/components/ui/Button";
import { Modal } from "@/components/ui/Modal";
import { FormField, Input, Textarea } from "@/components/ui/FormField";
import CopyableID from "@/components/ui/CopyableID";
import { useForm } from "react-hook-form";
import { useWarehouses, useCreateWarehouse, type Warehouse } from "@/lib/use-warehouses";
import { toastSuccess, toastError } from "@/lib/toast";

interface CreateFormValues {
  name: string;
  address: string;
}

export default function WarehousesPage() {
  const router = useRouter();
  const [page, setPage] = useState(1);
  const [sortField, setSortField] = useState("created_at");
  const [sortDesc, setSortDesc] = useState(true);
  const [searchInput, setSearchInput] = useState("");
  const [nameFilter, setNameFilter] = useState("");
  const [createOpen, setCreateOpen] = useState(false);

  const listQuery = useWarehouses({
    page,
    pageSize: 10,
    sortField,
    sortDesc,
    name: nameFilter || undefined,
  });

  const createWarehouse = useCreateWarehouse();

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<CreateFormValues>({ defaultValues: { name: "", address: "" } });

  const warehouses = listQuery.data?.data ?? [];
  const total = listQuery.data?.meta?.total ?? 0;

  const handleSort = useCallback((field: string, desc: boolean) => {
    setSortField(field);
    setSortDesc(desc);
    setPage(1);
  }, []);

  const handleSearch = useCallback(() => {
    setNameFilter(searchInput);
    setPage(1);
  }, [searchInput]);

  const clearSearch = useCallback(() => {
    setSearchInput("");
    setNameFilter("");
    setPage(1);
  }, []);

  const onCreateSubmit = (data: CreateFormValues) => {
    createWarehouse.mutate(
      { name: data.name, address: data.address || undefined },
      {
        onSuccess: () => {
          toastSuccess("Warehouse created");
          setCreateOpen(false);
          reset();
        },
        onError: toastError,
      },
    );
  };

  const columns: Column<Warehouse>[] = [
    {
      key: "id",
      header: "ID",
      render: (row) => <CopyableID id={row.id} label="Warehouse ID" />,
    },
    { key: "name", header: "Name", sortable: true },
    {
      key: "address",
      header: "Address",
      render: (row) =>
        row.address || <span className="text-gray-400">-</span>,
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

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Warehouses</h1>
        <Button onClick={() => setCreateOpen(true)}>
          <Plus className="h-4 w-4" />
          New Warehouse
        </Button>
      </div>

      {}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <input
            type="text"
            placeholder="Search by warehouse name..."
            value={searchInput}
            onChange={(e) => setSearchInput(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleSearch()}
            className="w-full rounded-md border border-gray-300 py-2 pl-9 pr-8 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          />
          <Search className="absolute left-3 top-2.5 h-4 w-4 text-gray-400" />
          {searchInput && (
            <button
              onClick={clearSearch}
              className="absolute right-2 top-2.5 text-gray-400 hover:text-gray-600"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
      </div>

      {}
      <DataTable<Warehouse>
        columns={columns}
        data={warehouses}
        total={total}
        page={page}
        pageSize={10}
        onPageChange={setPage}
        sortField={sortField}
        sortDesc={sortDesc}
        onSort={handleSort}
        loading={listQuery.isLoading}
        emptyMessage="No warehouses found"
        rowKey={(row) => row.id}
        onRowClick={(row) => router.push(`/warehouses/${row.id}`)}
      />

      {}
      <Modal
        open={createOpen}
        onClose={() => {
          setCreateOpen(false);
          reset();
        }}
        title="Create Warehouse"
        size="sm"
      >
        <form onSubmit={handleSubmit(onCreateSubmit)} className="space-y-4">
          <FormField label="Name" required error={errors.name}>
            <Input
              registration={register("name", { required: "Name is required" })}
              error={errors.name}
              placeholder="Warehouse name"
            />
          </FormField>
          <FormField label="Address">
            <Textarea
              registration={register("address")}
              placeholder="Warehouse address (optional)"
              rows={2}
            />
          </FormField>
          <div className="flex justify-end gap-3 pt-2">
            <Button
              type="button"
              variant="secondary"
              onClick={() => {
                setCreateOpen(false);
                reset();
              }}
            >
              Cancel
            </Button>
            <Button type="submit" loading={createWarehouse.isPending}>
              Create
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}
