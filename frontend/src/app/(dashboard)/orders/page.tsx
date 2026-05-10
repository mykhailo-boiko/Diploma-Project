"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Plus, Search, X } from "lucide-react";
import DataTable, { type Column } from "@/components/ui/DataTable";
import StatusBadge from "@/components/ui/StatusBadge";
import Button from "@/components/ui/Button";
import { useOrders, useOrderSearch, type Order } from "@/lib/use-orders";
import { safeFixed } from "@/lib/format";

const ORDER_STATUSES = [
  { value: "", label: "All statuses" },
  { value: "pending", label: "Pending" },
  { value: "confirmed", label: "Confirmed" },
  { value: "processing", label: "Processing" },
  { value: "shipped", label: "Shipped" },
  { value: "delivered", label: "Delivered" },
  { value: "completed", label: "Completed" },
  { value: "cancelled", label: "Cancelled" },
  { value: "returned", label: "Returned" },
];

const columns: Column<Order>[] = [
  {
    key: "id",
    header: "Order ID",
    render: (row) => (
      <span className="font-mono text-xs text-gray-600">
        {row.id.slice(0, 8)}...
      </span>
    ),
  },
  { key: "customer_name", header: "Customer", sortable: true },
  {
    key: "status",
    header: "Status",
    sortable: true,
    render: (row) => <StatusBadge status={row.status} />,
  },
  {
    key: "total_amount",
    header: "Total",
    sortable: true,
    render: (row) => `$${safeFixed(row.total_amount, 2)}`,
    className: "text-right",
  },
  {
    key: "created_at",
    header: "Created",
    sortable: true,
    render: (row) => new Date(row.created_at).toLocaleDateString(),
  },
];

export default function OrdersPage() {
  const router = useRouter();
  const [page, setPage] = useState(1);
  const [sortField, setSortField] = useState("created_at");
  const [sortDesc, setSortDesc] = useState(true);
  const [statusFilter, setStatusFilter] = useState("");
  const [searchQuery, setSearchQuery] = useState("");
  const [searchInput, setSearchInput] = useState("");

  const isSearching = searchQuery.length >= 2;

  const listQuery = useOrders({
    page,
    pageSize: 10,
    sortField,
    sortDesc,
    status: statusFilter || undefined,
  });

  const searchResult = useOrderSearch(searchQuery, page, 10);

  const activeQuery = isSearching ? searchResult : listQuery;
  const orders = activeQuery.data?.data ?? [];
  const total = activeQuery.data?.meta?.total ?? 0;

  const handleSort = useCallback((field: string, desc: boolean) => {
    setSortField(field);
    setSortDesc(desc);
    setPage(1);
  }, []);

  const handleSearch = useCallback(() => {
    setSearchQuery(searchInput);
    setPage(1);
  }, [searchInput]);

  const clearSearch = useCallback(() => {
    setSearchInput("");
    setSearchQuery("");
    setPage(1);
  }, []);

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Orders</h1>
        <Button onClick={() => router.push("/orders/new")}>
          <Plus className="h-4 w-4" />
          New Order
        </Button>
      </div>

      {}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <input
            type="text"
            placeholder="Search by customer or order ID..."
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

        {!isSearching && (
          <select
            value={statusFilter}
            onChange={(e) => {
              setStatusFilter(e.target.value);
              setPage(1);
            }}
            className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
          >
            {ORDER_STATUSES.map((s) => (
              <option key={s.value} value={s.value}>
                {s.label}
              </option>
            ))}
          </select>
        )}
      </div>

      {}
      <DataTable<Order>
        columns={columns}
        data={orders}
        total={total}
        page={page}
        pageSize={10}
        onPageChange={setPage}
        sortField={isSearching ? undefined : sortField}
        sortDesc={sortDesc}
        onSort={isSearching ? undefined : handleSort}
        loading={activeQuery.isLoading}
        emptyMessage={isSearching ? "No orders matching your search" : "No orders found"}
        rowKey={(row) => row.id}
        onRowClick={(row) => router.push(`/orders/${row.id}`)}
      />
    </div>
  );
}
