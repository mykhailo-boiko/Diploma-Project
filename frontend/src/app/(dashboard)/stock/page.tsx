"use client";

import { useState, useCallback } from "react";
import { AlertTriangle, ArrowDownUp, Package } from "lucide-react";
import DataTable, { type Column } from "@/components/ui/DataTable";
import StatusBadge from "@/components/ui/StatusBadge";
import Button from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Modal } from "@/components/ui/Modal";
import { FormField, Input, Select } from "@/components/ui/FormField";
import CopyableID from "@/components/ui/CopyableID";
import {
  useStock,
  useStockMovements,
  useLowStock,
  useAdjustStock,
  type Stock,
  type StockMovement,
  type LowStockItem,
} from "@/lib/use-stock";
import { toastSuccess, toastError } from "@/lib/toast";

type Tab = "levels" | "movements" | "low-stock";

const MOVEMENT_TYPES = [
  { value: "", label: "All types" },
  { value: "reserve", label: "Reserve" },
  { value: "release", label: "Release" },
  { value: "inbound", label: "Inbound" },
  { value: "outbound", label: "Outbound" },
  { value: "adjustment", label: "Adjustment" },
];

const ADJUST_TYPES = [
  { value: "inbound", label: "Inbound (+)" },
  { value: "outbound", label: "Outbound (-)" },
  { value: "adjustment", label: "Adjustment" },
];

export default function StockPage() {
  const [tab, setTab] = useState<Tab>("levels");

  const [stockPage, setStockPage] = useState(1);
  const [stockSort, setStockSort] = useState("updated_at");
  const [stockSortDesc, setStockSortDesc] = useState(true);
  const [warehouseFilter, setWarehouseFilter] = useState("");

  const [movPage, setMovPage] = useState(1);
  const [movSort, setMovSort] = useState("created_at");
  const [movSortDesc, setMovSortDesc] = useState(true);
  const [movTypeFilter, setMovTypeFilter] = useState("");

  const [adjustOpen, setAdjustOpen] = useState(false);
  const [adjustProductId, setAdjustProductId] = useState("");
  const [adjustWarehouseId, setAdjustWarehouseId] = useState("");
  const [adjustQty, setAdjustQty] = useState("");
  const [adjustType, setAdjustType] = useState("inbound");
  const [adjustRef, setAdjustRef] = useState("");

  const stockQuery = useStock({
    page: stockPage,
    pageSize: 10,
    sortField: stockSort,
    sortDesc: stockSortDesc,
    warehouseId: warehouseFilter || undefined,
  });

  const movQuery = useStockMovements({
    page: movPage,
    pageSize: 10,
    sortField: movSort,
    sortDesc: movSortDesc,
    type: movTypeFilter || undefined,
  });

  const lowStockQuery = useLowStock();
  const adjustStock = useAdjustStock();

  const handleStockSort = useCallback((f: string, d: boolean) => {
    setStockSort(f);
    setStockSortDesc(d);
    setStockPage(1);
  }, []);

  const handleMovSort = useCallback((f: string, d: boolean) => {
    setMovSort(f);
    setMovSortDesc(d);
    setMovPage(1);
  }, []);

  const handleAdjust = () => {
    if (!adjustProductId || !adjustWarehouseId || !adjustQty) return;
    adjustStock.mutate(
      {
        product_id: adjustProductId,
        warehouse_id: adjustWarehouseId,
        quantity: parseInt(adjustQty, 10),
        type: adjustType,
        reference: adjustRef || undefined,
      },
      {
        onSuccess: () => {
          toastSuccess("Stock adjusted successfully");
          setAdjustOpen(false);
          setAdjustProductId("");
          setAdjustWarehouseId("");
          setAdjustQty("");
          setAdjustType("inbound");
          setAdjustRef("");
        },
        onError: toastError,
      },
    );
  };

  const stockColumns: Column<Stock>[] = [
    {
      key: "product_id",
      header: "Product ID",
      sortable: true,
      render: (row) => <CopyableID id={row.product_id} label="Product ID" />,
    },
    {
      key: "warehouse_id",
      header: "Warehouse ID",
      sortable: true,
      render: (row) => <CopyableID id={row.warehouse_id} label="Warehouse ID" />,
    },
    {
      key: "quantity",
      header: "Quantity",
      sortable: true,
      className: "text-right",
    },
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
      sortable: true,
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

  const movColumns: Column<StockMovement>[] = [
    {
      key: "type",
      header: "Type",
      sortable: true,
      render: (row) => {
        const variant =
          row.type === "inbound"
            ? "success"
            : row.type === "outbound"
              ? "error"
              : row.type === "reserve"
                ? "warning"
                : row.type === "release"
                  ? "info"
                  : "neutral";
        return <StatusBadge status={row.type} variant={variant} />;
      },
    },
    {
      key: "quantity",
      header: "Quantity",
      sortable: true,
      className: "text-right",
      render: (row) => {
        const sign = row.type === "inbound" || row.type === "release" ? "+" : "-";
        return (
          <span
            className={
              sign === "+"
                ? "font-medium text-green-600"
                : "font-medium text-red-600"
            }
          >
            {sign}{row.quantity}
          </span>
        );
      },
    },
    {
      key: "product_id",
      header: "Product",
      render: (row) => <CopyableID id={row.product_id} label="Product ID" />,
    },
    {
      key: "warehouse_id",
      header: "Warehouse",
      render: (row) => <CopyableID id={row.warehouse_id} label="Warehouse ID" />,
    },
    {
      key: "reference",
      header: "Reference",
      render: (row) =>
        row.reference ? (
          <span className="text-xs text-gray-600">{row.reference}</span>
        ) : (
          <span className="text-gray-400">-</span>
        ),
    },
    {
      key: "created_at",
      header: "Date",
      sortable: true,
      render: (row) => new Date(row.created_at).toLocaleString(),
    },
  ];

  const lowStockColumns: Column<LowStockItem>[] = [
    { key: "product_sku", header: "SKU", render: (row) => <span className="font-mono text-xs">{row.product_sku}</span> },
    { key: "product_name", header: "Product" },
    {
      key: "product_id",
      header: "Product ID",
      render: (row) => <CopyableID id={row.product_id} label="Product ID" />,
    },
    {
      key: "warehouse_id",
      header: "Warehouse",
      render: (row) => <CopyableID id={row.warehouse_id} label="Warehouse ID" />,
    },
    { key: "available", header: "Available", className: "text-right", render: (row) => <span className="font-medium text-red-600">{row.available}</span> },
    { key: "min_threshold", header: "Threshold", className: "text-right" },
    {
      key: "quantity",
      header: "Total Qty",
      className: "text-right",
    },
    {
      key: "reserved",
      header: "Reserved",
      className: "text-right",
    },
  ];

  const tabs: { key: Tab; label: string; icon: typeof Package }[] = [
    { key: "levels", label: "Stock Levels", icon: Package },
    { key: "movements", label: "Movements", icon: ArrowDownUp },
    { key: "low-stock", label: "Low Stock", icon: AlertTriangle },
  ];

  const lowStockItems = lowStockQuery.data?.data ?? [];

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold text-gray-900">Stock</h1>
        <Button onClick={() => setAdjustOpen(true)}>
          <ArrowDownUp className="h-4 w-4" />
          Adjust Stock
        </Button>
      </div>

      {}
      <div className="flex gap-1 border-b border-gray-200">
        {tabs.map((t) => (
          <button
            key={t.key}
            onClick={() => setTab(t.key)}
            className={`flex items-center gap-1.5 border-b-2 px-4 py-2 text-sm font-medium transition-colors ${
              tab === t.key
                ? "border-blue-500 text-blue-600"
                : "border-transparent text-gray-500 hover:border-gray-300 hover:text-gray-700"
            }`}
          >
            <t.icon className="h-4 w-4" />
            {t.label}
            {t.key === "low-stock" && lowStockItems.length > 0 && (
              <span className="ml-1 inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-red-100 px-1.5 text-xs font-semibold text-red-700">
                {lowStockItems.length}
              </span>
            )}
          </button>
        ))}
      </div>

      {}
      {tab === "levels" && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-3">
            <input
              type="text"
              placeholder="Filter by warehouse ID..."
              value={warehouseFilter}
              onChange={(e) => {
                setWarehouseFilter(e.target.value);
                setStockPage(1);
              }}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </div>
          <DataTable<Stock>
            columns={stockColumns}
            data={stockQuery.data?.data ?? []}
            total={stockQuery.data?.meta?.total ?? 0}
            page={stockPage}
            pageSize={10}
            onPageChange={setStockPage}
            sortField={stockSort}
            sortDesc={stockSortDesc}
            onSort={handleStockSort}
            loading={stockQuery.isLoading}
            emptyMessage="No stock entries found"
            rowKey={(row) => row.id}
          />
        </div>
      )}

      {tab === "movements" && (
        <div className="space-y-3">
          <div className="flex flex-wrap items-center gap-3">
            <select
              value={movTypeFilter}
              onChange={(e) => {
                setMovTypeFilter(e.target.value);
                setMovPage(1);
              }}
              className="rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            >
              {MOVEMENT_TYPES.map((m) => (
                <option key={m.value} value={m.value}>
                  {m.label}
                </option>
              ))}
            </select>
          </div>
          <DataTable<StockMovement>
            columns={movColumns}
            data={movQuery.data?.data ?? []}
            total={movQuery.data?.meta?.total ?? 0}
            page={movPage}
            pageSize={10}
            onPageChange={setMovPage}
            sortField={movSort}
            sortDesc={movSortDesc}
            onSort={handleMovSort}
            loading={movQuery.isLoading}
            emptyMessage="No stock movements found"
            rowKey={(row) => row.id}
          />
        </div>
      )}

      {tab === "low-stock" && (
        <div className="space-y-3">
          {lowStockItems.length === 0 && !lowStockQuery.isLoading ? (
            <Card>
              <div className="flex flex-col items-center py-8 text-gray-500">
                <Package className="mb-2 h-10 w-10 text-gray-300" />
                <p className="font-medium">All stock levels are healthy</p>
                <p className="text-sm">No products below their minimum threshold</p>
              </div>
            </Card>
          ) : (
            <DataTable<LowStockItem>
              columns={lowStockColumns}
              data={lowStockItems}
              total={lowStockItems.length}
              page={1}
              pageSize={100}
              onPageChange={() => {}}
              loading={lowStockQuery.isLoading}
              emptyMessage="No low stock items"
              rowKey={(row) => row.id}
            />
          )}
        </div>
      )}

      {}
      <Modal
        open={adjustOpen}
        onClose={() => setAdjustOpen(false)}
        title="Adjust Stock"
        size="md"
      >
        <div className="space-y-4">
          <FormField label="Product ID" required>
            <input
              value={adjustProductId}
              onChange={(e) => setAdjustProductId(e.target.value)}
              placeholder="Product UUID"
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </FormField>
          <FormField label="Warehouse ID" required>
            <input
              value={adjustWarehouseId}
              onChange={(e) => setAdjustWarehouseId(e.target.value)}
              placeholder="Warehouse UUID"
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </FormField>
          <div className="grid grid-cols-2 gap-4">
            <FormField label="Type" required>
              <select
                value={adjustType}
                onChange={(e) => setAdjustType(e.target.value)}
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              >
                {ADJUST_TYPES.map((t) => (
                  <option key={t.value} value={t.value}>
                    {t.label}
                  </option>
                ))}
              </select>
            </FormField>
            <FormField label="Quantity" required>
              <input
                type="number"
                value={adjustQty}
                onChange={(e) => setAdjustQty(e.target.value)}
                placeholder="0"
                min={1}
                className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
            </FormField>
          </div>
          <FormField label="Reference">
            <input
              value={adjustRef}
              onChange={(e) => setAdjustRef(e.target.value)}
              placeholder="Optional reference note"
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </FormField>
          <div className="flex justify-end gap-3 pt-2">
            <Button variant="secondary" onClick={() => setAdjustOpen(false)}>
              Cancel
            </Button>
            <Button
              onClick={handleAdjust}
              loading={adjustStock.isPending}
              disabled={!adjustProductId || !adjustWarehouseId || !adjustQty}
            >
              Apply Adjustment
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}
