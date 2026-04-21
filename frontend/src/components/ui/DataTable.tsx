"use client";

import { useCallback } from "react";
import clsx from "clsx";
import { ChevronUp, ChevronDown, ChevronsUpDown, Loader2 } from "lucide-react";

export interface Column<T> {
  key: string;
  header: string;
  sortable?: boolean;
  render?: (row: T) => React.ReactNode;
  className?: string;
}

export interface DataTableProps<T> {
  columns: Column<T>[];
  data: T[];
  total?: number;
  page?: number;
  pageSize?: number;
  onPageChange?: (page: number) => void;
  sortField?: string;
  sortDesc?: boolean;
  onSort?: (field: string, desc: boolean) => void;
  loading?: boolean;
  emptyMessage?: string;
  rowKey?: (row: T) => string;
  onRowClick?: (row: T) => void;
}

export default function DataTable<T>({
  columns,
  data,
  total = 0,
  page = 1,
  pageSize = 10,
  onPageChange,
  sortField,
  sortDesc = false,
  onSort,
  loading = false,
  emptyMessage = "No data found",
  rowKey,
  onRowClick,
}: DataTableProps<T>) {
  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  const handleSort = useCallback(
    (field: string) => {
      if (!onSort) return;
      if (sortField === field) {
        onSort(field, !sortDesc);
      } else {
        onSort(field, false);
      }
    },
    [onSort, sortField, sortDesc],
  );

  const SortIcon = ({ field }: { field: string }) => {
    if (sortField !== field)
      return <ChevronsUpDown className="ml-1 inline h-3.5 w-3.5 text-gray-400" />;
    return sortDesc ? (
      <ChevronDown className="ml-1 inline h-3.5 w-3.5 text-gray-700" />
    ) : (
      <ChevronUp className="ml-1 inline h-3.5 w-3.5 text-gray-700" />
    );
  };

  return (
    <div className="overflow-hidden rounded-lg border border-gray-200 bg-white">
      <div className="overflow-x-auto">
        <table className="min-w-full divide-y divide-gray-200">
          <thead className="bg-gray-50">
            <tr>
              {columns.map((col) => (
                <th
                  key={col.key}
                  className={clsx(
                    "px-4 py-3 text-left text-xs font-medium uppercase tracking-wider text-gray-500",
                    col.sortable && "cursor-pointer select-none hover:text-gray-700",
                    col.className,
                  )}
                  onClick={col.sortable ? () => handleSort(col.key) : undefined}
                >
                  {col.header}
                  {col.sortable && <SortIcon field={col.key} />}
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-gray-200 bg-white">
            {loading ? (
              <tr>
                <td colSpan={columns.length} className="px-4 py-12 text-center">
                  <Loader2 className="mx-auto h-6 w-6 animate-spin text-gray-400" />
                  <p className="mt-2 text-sm text-gray-500">Loading...</p>
                </td>
              </tr>
            ) : data.length === 0 ? (
              <tr>
                <td
                  colSpan={columns.length}
                  className="px-4 py-12 text-center text-sm text-gray-500"
                >
                  {emptyMessage}
                </td>
              </tr>
            ) : (
              data.map((row, i) => (
                <tr
                  key={rowKey ? rowKey(row) : i}
                  className={clsx(
                    "transition-colors hover:bg-gray-50",
                    onRowClick && "cursor-pointer",
                  )}
                  onClick={onRowClick ? () => onRowClick(row) : undefined}
                >
                  {columns.map((col) => (
                    <td
                      key={col.key}
                      className={clsx(
                        "whitespace-nowrap px-4 py-3 text-sm text-gray-900",
                        col.className,
                      )}
                    >
                      {col.render
                        ? col.render(row)
                        : String((row as Record<string, unknown>)[col.key] ?? "")}
                    </td>
                  ))}
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {total > 0 && onPageChange && (
        <div className="flex items-center justify-between border-t border-gray-200 bg-white px-4 py-3">
          <p className="text-sm text-gray-700">
            Showing{" "}
            <span className="font-medium">{(page - 1) * pageSize + 1}</span> to{" "}
            <span className="font-medium">
              {Math.min(page * pageSize, total)}
            </span>{" "}
            of <span className="font-medium">{total}</span> results
          </p>
          <div className="flex gap-1">
            <button
              onClick={() => onPageChange(page - 1)}
              disabled={page <= 1}
              className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Previous
            </button>
            {Array.from({ length: Math.min(5, totalPages) }, (_, i) => {
              let pageNum: number;
              if (totalPages <= 5) {
                pageNum = i + 1;
              } else if (page <= 3) {
                pageNum = i + 1;
              } else if (page >= totalPages - 2) {
                pageNum = totalPages - 4 + i;
              } else {
                pageNum = page - 2 + i;
              }
              return (
                <button
                  key={pageNum}
                  onClick={() => onPageChange(pageNum)}
                  className={clsx(
                    "rounded-md border px-3 py-1.5 text-sm font-medium",
                    pageNum === page
                      ? "border-blue-500 bg-blue-50 text-blue-600"
                      : "border-gray-300 bg-white text-gray-700 hover:bg-gray-50",
                  )}
                >
                  {pageNum}
                </button>
              );
            })}
            <button
              onClick={() => onPageChange(page + 1)}
              disabled={page >= totalPages}
              className="rounded-md border border-gray-300 bg-white px-3 py-1.5 text-sm font-medium text-gray-700 hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              Next
            </button>
          </div>
        </div>
      )}
    </div>
  );
}
