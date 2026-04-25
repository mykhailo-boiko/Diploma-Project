"use client";

import { useState, useCallback } from "react";
import { useRouter } from "next/navigation";
import { Plus, Search, X, Trash2 } from "lucide-react";
import DataTable, { type Column } from "@/components/ui/DataTable";
import StatusBadge from "@/components/ui/StatusBadge";
import Button from "@/components/ui/Button";
import { ConfirmDialog } from "@/components/ui/Modal";
import { useProducts, useDeleteProduct, type Product } from "@/lib/use-products";
import { toastSuccess, toastError } from "@/lib/toast";

const CATEGORIES = [
  { value: "", label: "All categories" },
  { value: "Electronics", label: "Electronics" },
  { value: "Furniture", label: "Furniture" },
  { value: "Clothing", label: "Clothing" },
  { value: "Food & Beverage", label: "Food & Beverage" },
  { value: "Tools", label: "Tools" },
];

export default function ProductsPage() {
  const router = useRouter();
  const [page, setPage] = useState(1);
  const [sortField, setSortField] = useState("created_at");
  const [sortDesc, setSortDesc] = useState(true);
  const [categoryFilter, setCategoryFilter] = useState("");
  const [searchInput, setSearchInput] = useState("");
  const [nameFilter, setNameFilter] = useState("");
  const [deleteTarget, setDeleteTarget] = useState<Product | null>(null);

  const listQuery = useProducts({
    page,
    pageSize: 10,
    sortField,
    sortDesc,
    category: categoryFilter || undefined,
    name: nameFilter || undefined,
  });

  const deleteProduct = useDeleteProduct();

  const products = listQuery.data?.data ?? [];
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

  const handleDelete = () => {
    if (!deleteTarget) return;
    deleteProduct.mutate(
      { id: deleteTarget.id },
      {
        onSuccess: () => {
          toastSuccess(`Product "${deleteTarget.name}" deleted`);
          setDeleteTarget(null);
        },
        onError: (err) => {
          toastError(err);
          setDeleteTarget(null);
        },
      },
    );
  };

  const columns: Column<Product>[] = [
    {
      key: "sku",
      header: "SKU",
      sortable: true,
      render: (row) => (
        <span className="font-mono text-xs text-gray-600">{row.sku}</span>
      ),
    },
    { key: "name", header: "Name", sortable: true },
    {
      key: "category",
      header: "Category",
      sortable: true,
      render: (row) =>
        row.category ? (
          <StatusBadge status={row.category} variant="neutral" />
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      key: "unit_price",
      header: "Unit Price",
      sortable: true,
      render: (row) => `$${row.unit_price.toFixed(2)}`,
      className: "text-right",
    },
    {
      key: "created_at",
      header: "Created",
      sortable: true,
      render: (row) => new Date(row.created_at).toLocaleDateString(),
    },
    {
      key: "actions" as keyof Product,
      header: "",
      render: (row) => (
        <button
          onClick={(e) => {
            e.stopPropagation();
            setDeleteTarget(row);
          }}
          className="rounded-md p-1.5 text-gray-400 hover:bg-red-50 hover:text-red-600"
        >
          <Trash2 className="h-4 w-4" />
        </button>
      ),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Products</h1>
        <Button onClick={() => router.push("/products/new")}>
          <Plus className="h-4 w-4" />
          New Product
        </Button>
      </div>

      {}
      <div className="flex flex-wrap items-center gap-3">
        <div className="relative flex-1 min-w-[200px] max-w-md">
          <input
            type="text"
            placeholder="Search by product name..."
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

        <select
          value={categoryFilter}
          onChange={(e) => {
            setCategoryFilter(e.target.value);
            setPage(1);
          }}
          className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
        >
          {CATEGORIES.map((c) => (
            <option key={c.value} value={c.value}>
              {c.label}
            </option>
          ))}
        </select>
      </div>

      {}
      <DataTable<Product>
        columns={columns}
        data={products}
        total={total}
        page={page}
        pageSize={10}
        onPageChange={setPage}
        sortField={sortField}
        sortDesc={sortDesc}
        onSort={handleSort}
        loading={listQuery.isLoading}
        emptyMessage="No products found"
        rowKey={(row) => row.id}
        onRowClick={(row) => router.push(`/products/${row.id}`)}
      />

      {}
      <ConfirmDialog
        open={!!deleteTarget}
        onClose={() => setDeleteTarget(null)}
        onConfirm={handleDelete}
        title="Delete Product"
        message={`Are you sure you want to delete "${deleteTarget?.name}"? This action cannot be undone.`}
        confirmLabel="Delete"
        variant="danger"
        loading={deleteProduct.isPending}
      />
    </div>
  );
}
