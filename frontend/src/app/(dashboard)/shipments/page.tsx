"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import DataTable, { type Column } from "@/components/ui/DataTable";
import StatusBadge from "@/components/ui/StatusBadge";
import { useShipments, type Shipment } from "@/lib/use-shipments";

const SHIPMENT_STATUSES = [
  { value: "", label: "All statuses" },
  { value: "created", label: "Created" },
  { value: "picked_up", label: "Picked Up" },
  { value: "in_transit", label: "In Transit" },
  { value: "delivered", label: "Delivered" },
  { value: "failed", label: "Failed" },
  { value: "returned", label: "Returned" },
];

const columns: Column<Shipment>[] = [
  {
    key: "id",
    header: "Shipment ID",
    render: (row) => (
      <span className="font-mono text-xs text-gray-600">
        {row.id.slice(0, 8)}...
      </span>
    ),
  },
  {
    key: "order_id",
    header: "Order ID",
    render: (row) => (
      <span className="font-mono text-xs text-gray-600">
        {row.order_id.slice(0, 8)}...
      </span>
    ),
  },
  {
    key: "status",
    header: "Status",
    sortable: true,
    render: (row) => <StatusBadge status={row.status} />,
  },
  {
    key: "address",
    header: "Address",
    render: (row) => (
      <span className="max-w-[200px] truncate block">{row.address}</span>
    ),
  },
  {
    key: "created_at",
    header: "Created",
    sortable: true,
    render: (row) => new Date(row.created_at).toLocaleDateString(),
  },
];

export default function ShipmentsPage() {
  const router = useRouter();
  const [page, setPage] = useState(1);
  const [sortField, setSortField] = useState("created_at");
  const [sortDesc, setSortDesc] = useState(true);
  const [statusFilter, setStatusFilter] = useState("");
  const [carrierFilter, setCarrierFilter] = useState("");

  const { data, isLoading } = useShipments({
    page,
    pageSize: 10,
    sortField,
    sortDesc,
    status: statusFilter || undefined,
    carrierId: carrierFilter || undefined,
  });

  const shipments = data?.data ?? [];
  const total = data?.meta?.total ?? 0;

  const handleSort = useCallback((field: string, desc: boolean) => {
    setSortField(field);
    setSortDesc(desc);
    setPage(1);
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Shipments</h1>
      </div>

      {}
      <div className="flex flex-wrap items-center gap-3">
        <select
          value={statusFilter}
          onChange={(e) => {
            setStatusFilter(e.target.value);
            setPage(1);
          }}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          {SHIPMENT_STATUSES.map((s) => (
            <option key={s.value} value={s.value}>
              {s.label}
            </option>
          ))}
        </select>

        <input
          type="text"
          placeholder="Filter by carrier ID..."
          value={carrierFilter}
          onChange={(e) => {
            setCarrierFilter(e.target.value);
            setPage(1);
          }}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        />
      </div>

      {}
      <DataTable<Shipment>
        columns={columns}
        data={shipments}
        total={total}
        page={page}
        pageSize={10}
        onPageChange={setPage}
        sortField={sortField}
        sortDesc={sortDesc}
        onSort={handleSort}
        loading={isLoading}
        emptyMessage="No shipments found"
        rowKey={(row) => row.id}
        onRowClick={(row) => router.push(`/shipments/${row.id}`)}
      />
    </div>
  );
}
