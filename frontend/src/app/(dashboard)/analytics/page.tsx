"use client";

import { useState, useMemo } from "react";
import { Card } from "@/components/ui";
import Button from "@/components/ui/Button";
import StatusBadge from "@/components/ui/StatusBadge";
import { Modal } from "@/components/ui/Modal";
import { FormField, Select } from "@/components/ui/FormField";
import { useForm } from "react-hook-form";
import { toastSuccess, toastError } from "@/lib/toast";
import { safeFixed, safeLocale } from "@/lib/format";
import {
  useSalesSummary,
  useSalesTrends,
  useInventorySummary,
  useLogisticsPerformance,
  useAnomalies,
  useOptimizations,
  useDownloadReport,
  type SalesTrend,
  type Anomaly,
  type Optimization,
  type ReportRequest,
} from "@/lib/use-analytics";
import {
  ResponsiveContainer,
  LineChart,
  Line,
  BarChart,
  Bar,
  PieChart,
  Pie,
  Cell,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  Legend,
} from "recharts";
import {
  TrendingUp,
  Package,
  Truck,
  AlertTriangle,
  FileText,
  Lightbulb,
  DollarSign,
  ShoppingCart,
  BarChart3,
  Info,
  Clock,
} from "lucide-react";

const PIE_COLORS = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6"];

function fmtCurrency(n: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 0,
  }).format(n);
}

function defaultDateRange() {
  const to = new Date();
  const from = new Date();
  from.setDate(from.getDate() - 30);
  const fmt = (d: Date) => d.toISOString().slice(0, 10);
  return { dateFrom: fmt(from), dateTo: fmt(to) };
}

const REPORT_TYPES = [
  { value: "sales", label: "Sales" },
  { value: "inventory", label: "Inventory" },
  { value: "logistics", label: "Logistics" },
  { value: "full", label: "Full Report" },
];

export default function AnalyticsPage() {
  const [reportModalOpen, setReportModalOpen] = useState(false);
  const { dateFrom, dateTo } = useMemo(() => defaultDateRange(), []);

  const salesSummary = useSalesSummary(dateFrom, dateTo);
  const salesTrends = useSalesTrends(dateFrom, dateTo, "day");
  const inventorySummary = useInventorySummary(dateFrom, dateTo);
  const logisticsPerf = useLogisticsPerformance(dateFrom, dateTo);
  const anomalies = useAnomalies(dateFrom, dateTo);
  const optimizations = useOptimizations(dateFrom, dateTo);
  const downloadReport = useDownloadReport();

  const sales = salesSummary.data?.data;
  const trends = salesTrends.data?.data ?? [];
  const inventory = inventorySummary.data?.data;
  const logistics = logisticsPerf.data?.data;
  const anomalyList = anomalies.data?.data ?? [];
  const optimizationList = optimizations.data?.data ?? [];

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-gray-900">Analytics</h1>
          <p className="mt-1 text-sm text-gray-500">
            Data from {dateFrom} to {dateTo}
          </p>
        </div>
        <Button onClick={() => setReportModalOpen(true)}>
          <FileText className="mr-1 h-4 w-4" />
          Generate Report
        </Button>
      </div>

      {}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <KPICard
          icon={<DollarSign className="h-5 w-5 text-green-600" />}
          title="Revenue"
          value={sales ? fmtCurrency(sales.total_revenue) : "-"}
          subtitle={sales ? `${sales.order_count} orders` : ""}
        />
        <KPICard
          icon={<ShoppingCart className="h-5 w-5 text-blue-600" />}
          title="Avg Order Value"
          value={sales ? fmtCurrency(sales.avg_order_value) : "-"}
          subtitle="per order"
        />
        <KPICard
          icon={<Package className="h-5 w-5 text-purple-600" />}
          title="Low Stock Items"
          value={inventory?.low_stock_count?.toString() ?? "-"}
          subtitle={`of ${inventory?.total_products ?? 0} products`}
        />
        <KPICard
          icon={<Truck className="h-5 w-5 text-orange-600" />}
          title="On-Time Rate"
          value={
            logistics ? `${safeFixed(logistics.on_time_rate, 1)}%` : "-"
          }
          subtitle={`${logistics?.delivered_count ?? 0} delivered`}
        />
      </div>

      {}
      <Card>
        <div className="p-5">
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900">
            <TrendingUp className="h-5 w-5 text-blue-500" />
            Sales Trends (30 Days)
          </h2>
          {trends.length > 0 ? (
            <SalesTrendsChart data={trends} />
          ) : (
            <EmptyState message="No sales data available for this period" />
          )}
        </div>
      </Card>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {}
        <Card>
          <div className="p-5">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900">
              <Package className="h-5 w-5 text-purple-500" />
              Inventory Health
            </h2>
            {inventory ? (
              <InventoryChart data={inventory} />
            ) : (
              <EmptyState message="No inventory data available" />
            )}
          </div>
        </Card>

        {}
        <Card>
          <div className="p-5">
            <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900">
              <Truck className="h-5 w-5 text-orange-500" />
              Logistics Performance
            </h2>
            {logistics ? (
              <LogisticsChart data={logistics} />
            ) : (
              <EmptyState message="No logistics data available" />
            )}
          </div>
        </Card>
      </div>

      {}
      <Card>
        <div className="p-5">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900">
              <AlertTriangle className="h-5 w-5 text-yellow-500" />
              Anomaly Alerts
              {anomalyList.length > 0 && (
                <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">
                  {anomalyList.length}
                </span>
              )}
            </h2>
          </div>
          <AnomalyExplainer />
          {anomalyList.length > 0 ? (
            <div className="grid gap-3 sm:grid-cols-2">
              {anomalyList.map((a, i) => (
                <AnomalyCard key={i} anomaly={a} />
              ))}
            </div>
          ) : (
            <EmptyState message="No anomalies detected" />
          )}
        </div>
      </Card>

      {}
      <Card>
        <div className="p-5">
          <div className="mb-2 flex items-center justify-between gap-3">
            <h2 className="flex items-center gap-2 text-lg font-semibold text-gray-900">
              <Lightbulb className="h-5 w-5 text-amber-500" />
              Reorder Recommendations
              {optimizationList.length > 0 && (
                <span className="rounded-full bg-amber-100 px-2 py-0.5 text-xs font-medium text-amber-700">
                  {optimizationList.length}
                </span>
              )}
            </h2>
          </div>
          <p className="mb-3 text-xs text-gray-500">
            Per-SKU reorder list: products whose available stock is below their
            minimum threshold. Demand is computed from outbound stock movements
            over the last 30 days; lead time is assumed 7 days with a 1.5×
            safety stock multiplier.
          </p>
          {optimizationList.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-gray-500">
                    <th className="pb-2 pr-3 font-medium">Urgency</th>
                    <th className="pb-2 pr-3 font-medium">Product</th>
                    <th className="pb-2 pr-3 font-medium">SKU</th>
                    <th className="pb-2 pr-3 font-medium">Warehouse</th>
                    <th className="pb-2 pr-3 text-right font-medium">Stock</th>
                    <th className="pb-2 pr-3 text-right font-medium">Threshold</th>
                    <th className="pb-2 pr-3 text-right font-medium">Avg/day</th>
                    <th className="pb-2 pr-3 text-right font-medium">Reorder Pt</th>
                    <th className="pb-2 pr-3 text-right font-medium">Order Qty</th>
                    <th className="pb-2 pr-3 text-right font-medium">Days Left</th>
                    <th className="pb-2 font-medium">Why</th>
                  </tr>
                </thead>
                <tbody>
                  {optimizationList.map((opt, i) => (
                    <OptimizationRow key={i} opt={opt} />
                  ))}
                </tbody>
              </table>
            </div>
          ) : (
            <EmptyState message="No reorder recommendations — all stock levels above threshold" />
          )}
        </div>
      </Card>

      {}
      <ReportModal
        open={reportModalOpen}
        onClose={() => setReportModalOpen(false)}
        onSubmit={(values) => {
          downloadReport.mutate(values, {
            onSuccess: (result) => {
              toastSuccess(`Downloaded ${result.filename}`);
              setReportModalOpen(false);
            },
            onError: toastError,
          });
        }}
        loading={downloadReport.isPending}
      />
    </div>
  );
}

function KPICard({
  icon,
  title,
  value,
  subtitle,
}: {
  icon: React.ReactNode;
  title: string;
  value: string;
  subtitle: string;
}) {
  return (
    <Card>
      <div className="flex items-start gap-3 p-4">
        <div className="rounded-lg bg-gray-50 p-2">{icon}</div>
        <div>
          <p className="text-sm text-gray-500">{title}</p>
          <p className="text-xl font-bold text-gray-900">{value}</p>
          <p className="text-xs text-gray-400">{subtitle}</p>
        </div>
      </div>
    </Card>
  );
}

function SalesTrendsChart({ data }: { data: SalesTrend[] }) {
  const chartData = data.map((d) => ({
    ...d,
    period: d.period.slice(5),
  }));

  return (
    <ResponsiveContainer width="100%" height={300}>
      <LineChart data={chartData}>
        <CartesianGrid strokeDasharray="3 3" />
        <XAxis dataKey="period" fontSize={12} />
        <YAxis yAxisId="left" fontSize={12} />
        <YAxis yAxisId="right" orientation="right" fontSize={12} />
        <Tooltip />
        <Legend />
        <Line
          yAxisId="left"
          type="monotone"
          dataKey="total_revenue"
          stroke="#3b82f6"
          name="Revenue"
          strokeWidth={2}
          dot={false}
        />
        <Line
          yAxisId="right"
          type="monotone"
          dataKey="total_orders"
          stroke="#10b981"
          name="Orders"
          strokeWidth={2}
          dot={false}
        />
      </LineChart>
    </ResponsiveContainer>
  );
}

function InventoryChart({
  data,
}: {
  data: { total_quantity: number; total_reserved: number; total_available: number; low_stock_count: number; turnover_rate: number };
}) {
  const pieData = [
    { name: "Available", value: data.total_available },
    { name: "Reserved", value: data.total_reserved },
  ];

  return (
    <div className="flex flex-col items-center gap-4 sm:flex-row">
      <ResponsiveContainer width="100%" height={200}>
        <PieChart>
          <Pie
            data={pieData}
            cx="50%"
            cy="50%"
            innerRadius={50}
            outerRadius={80}
            dataKey="value"
            label={({ name, percent }) =>
              `${name} ${((percent ?? 0) * 100).toFixed(0)}%`
            }
          >
            {pieData.map((_, idx) => (
              <Cell key={idx} fill={PIE_COLORS[idx]} />
            ))}
          </Pie>
          <Tooltip />
        </PieChart>
      </ResponsiveContainer>
      <div className="space-y-2 text-sm">
        <div>
          <span className="text-gray-500">Total Qty:</span>{" "}
          <span className="font-medium">{safeLocale(data.total_quantity)}</span>
        </div>
        <div>
          <span className="text-gray-500">Low Stock:</span>{" "}
          <span className="font-medium text-red-600">{data.low_stock_count}</span>
        </div>
        <div>
          <span className="text-gray-500">Turnover:</span>{" "}
          <span className="font-medium">{safeFixed(data.turnover_rate, 2)}</span>
        </div>
      </div>
    </div>
  );
}

function LogisticsChart({
  data,
}: {
  data: { total_shipments: number; delivered_count: number; failed_count: number; on_time_rate: number; avg_delivery_h: number };
}) {
  const barData = [
    { name: "Total", value: data.total_shipments, fill: "#3b82f6" },
    { name: "Delivered", value: data.delivered_count, fill: "#10b981" },
    { name: "Failed", value: data.failed_count, fill: "#ef4444" },
  ];

  return (
    <div className="space-y-4">
      <ResponsiveContainer width="100%" height={200}>
        <BarChart data={barData}>
          <CartesianGrid strokeDasharray="3 3" />
          <XAxis dataKey="name" fontSize={12} />
          <YAxis fontSize={12} />
          <Tooltip />
          <Bar dataKey="value" radius={[4, 4, 0, 0]}>
            {barData.map((entry, idx) => (
              <Cell key={idx} fill={entry.fill} />
            ))}
          </Bar>
        </BarChart>
      </ResponsiveContainer>
      <div className="flex gap-6 text-sm">
        <div>
          <span className="text-gray-500">On-Time:</span>{" "}
          <span className="font-medium text-green-600">
            {safeFixed(data.on_time_rate, 1)}%
          </span>
        </div>
        <div>
          <span className="text-gray-500">Avg Delivery:</span>{" "}
          <span className="font-medium">{safeFixed(data.avg_delivery_h, 1)}h</span>
        </div>
      </div>
    </div>
  );
}

const CATEGORY_STYLES: Record<string, { label: string; className: string }> = {
  sales: { label: "Sales", className: "bg-blue-100 text-blue-700" },
  logistics: { label: "Logistics", className: "bg-orange-100 text-orange-700" },
  inventory: { label: "Inventory", className: "bg-purple-100 text-purple-700" },
  business: { label: "Business", className: "bg-emerald-100 text-emerald-700" },
};

function AnomalyCard({ anomaly }: { anomaly: Anomaly }) {
  const cat = CATEGORY_STYLES[anomaly.category ?? ""] ?? {
    label: anomaly.category ?? "Other",
    className: "bg-gray-100 text-gray-700",
  };
  return (
    <div
      className={`rounded-lg border p-3 ${
        anomaly.severity === "critical"
          ? "border-red-200 bg-red-50"
          : "border-yellow-200 bg-yellow-50"
      }`}
    >
      <div className="mb-1 flex items-center justify-between gap-2">
        <div className="flex items-center gap-2">
          <span
            className={`rounded-full px-2 py-0.5 text-xs font-medium ${cat.className}`}
          >
            {cat.label}
          </span>
          <span className="text-sm font-medium text-gray-900">{anomaly.metric}</span>
        </div>
        <StatusBadge
          status={anomaly.severity === "critical" ? "failed" : "pending"}
        />
      </div>
      <p className="text-sm text-gray-600">{anomaly.message}</p>
      <div className="mt-1 flex flex-wrap gap-4 text-xs text-gray-400">
        <span>Value: {safeFixed(anomaly.value, 2)}</span>
        <span>Threshold: {safeFixed(anomaly.threshold, 2)}</span>
        <span>{anomaly.date}</span>
      </div>
    </div>
  );
}

function AnomalyExplainer() {
  const [open, setOpen] = useState(false);
  return (
    <div className="mb-3 rounded-lg border border-blue-100 bg-blue-50/60">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-2 px-3 py-2 text-left text-sm font-medium text-blue-900 hover:bg-blue-50"
      >
        <span className="flex items-center gap-2">
          <Info className="h-4 w-4" />
          How are these anomalies detected?
        </span>
        <span className="text-xs text-blue-600">{open ? "Hide" : "Show"}</span>
      </button>
      {open && (
        <div className="space-y-2 border-t border-blue-100 px-3 py-3 text-xs text-gray-700">
          <p>
            The detector scans aggregated daily metrics for the selected period
            and flags points that breach statistical or business thresholds.
            Severity is <span className="font-semibold">warning</span> at
            ~2σ deviation, <span className="font-semibold">critical</span> at
            ~3σ.
          </p>
          <table className="w-full border-separate border-spacing-y-1">
            <thead>
              <tr className="text-left text-gray-500">
                <th className="pr-3 font-medium">Category</th>
                <th className="pr-3 font-medium">Rule</th>
                <th className="font-medium">Source</th>
              </tr>
            </thead>
            <tbody className="text-gray-700">
              <tr>
                <td className="pr-3 align-top">
                  <span className="rounded-full bg-blue-100 px-2 py-0.5 font-medium text-blue-700">
                    Sales
                  </span>
                </td>
                <td className="pr-3">
                  Daily revenue deviates &gt;2σ from period mean, OR a day with
                  zero orders.
                </td>
                <td>
                  <code>analytics.sales_daily</code>
                </td>
              </tr>
              <tr>
                <td className="pr-3 align-top">
                  <span className="rounded-full bg-orange-100 px-2 py-0.5 font-medium text-orange-700">
                    Logistics
                  </span>
                </td>
                <td className="pr-3">
                  Failure rate &gt;20% per day, OR on-time delivery rate &lt;80%.
                </td>
                <td>
                  <code>analytics.logistics_daily</code>
                </td>
              </tr>
              <tr>
                <td className="pr-3 align-top">
                  <span className="rounded-full bg-purple-100 px-2 py-0.5 font-medium text-purple-700">
                    Inventory
                  </span>
                </td>
                <td className="pr-3">
                  More than 10% of products below their minimum stock threshold.
                </td>
                <td>
                  <code>analytics.inventory_snapshots</code>
                </td>
              </tr>
              <tr>
                <td className="pr-3 align-top">
                  <span className="rounded-full bg-emerald-100 px-2 py-0.5 font-medium text-emerald-700">
                    Business
                  </span>
                </td>
                <td className="pr-3">
                  Average order value drops &gt;2σ below the period mean
                  (requires ≥7 sample days).
                </td>
                <td>
                  <code>analytics.sales_daily</code>
                </td>
              </tr>
            </tbody>
          </table>
          <p className="text-gray-500">
            Aggregates are kept up-to-date by NATS consumers reacting to
            order/shipment/inventory events.
          </p>
        </div>
      )}
    </div>
  );
}

function OptimizationRow({ opt }: { opt: Optimization }) {
  const urgencyClass =
    opt.urgency === "critical"
      ? "bg-red-100 text-red-700"
      : opt.urgency === "warning"
        ? "bg-amber-100 text-amber-700"
        : "bg-blue-100 text-blue-700";

  const daysLeft =
    !isFinite(opt.days_until_stockout) || opt.days_until_stockout >= 999
      ? "—"
      : safeFixed(opt.days_until_stockout, 1);

  return (
    <tr className="border-b last:border-0 align-top">
      <td className="py-2 pr-3">
        <span
          className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium uppercase ${urgencyClass}`}
        >
          {opt.urgency === "critical" && <AlertTriangle className="h-3 w-3" />}
          {opt.urgency === "warning" && <Clock className="h-3 w-3" />}
          {opt.urgency}
        </span>
      </td>
      <td className="py-2 pr-3 font-medium text-gray-900">{opt.product_name}</td>
      <td className="py-2 pr-3 font-mono text-xs text-gray-500">{opt.product_sku}</td>
      <td className="py-2 pr-3 text-gray-700">{opt.warehouse_name}</td>
      <td className="py-2 pr-3 text-right font-medium text-gray-900">{opt.current_stock}</td>
      <td className="py-2 pr-3 text-right text-gray-500">{opt.min_threshold}</td>
      <td className="py-2 pr-3 text-right text-gray-500">{safeFixed(opt.avg_daily_demand, 1)}</td>
      <td className="py-2 pr-3 text-right text-gray-700">{safeFixed(opt.reorder_point, 0)}</td>
      <td className="py-2 pr-3 text-right font-semibold text-blue-600">{safeFixed(opt.recommended_order, 0)}</td>
      <td className="py-2 pr-3 text-right text-gray-700">{daysLeft}</td>
      <td className="py-2 max-w-md text-xs text-gray-500">{opt.reason}</td>
    </tr>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex h-32 items-center justify-center">
      <div className="text-center text-sm text-gray-400">
        <BarChart3 className="mx-auto mb-2 h-8 w-8" />
        {message}
      </div>
    </div>
  );
}

const REPORT_FORMATS = [
  { value: "csv", label: "CSV (Excel-friendly)" },
  { value: "json", label: "JSON (raw data)" },
];

function ReportModal({
  open,
  onClose,
  onSubmit,
  loading,
}: {
  open: boolean;
  onClose: () => void;
  onSubmit: (values: ReportRequest) => void;
  loading: boolean;
}) {
  const { dateFrom, dateTo } = defaultDateRange();
  const { register, handleSubmit, reset } = useForm<ReportRequest>({
    defaultValues: {
      report_type: "full",
      date_from: dateFrom,
      date_to: dateTo,
      format: "csv",
    },
  });

  const handleClose = () => {
    reset();
    onClose();
  };

  return (
    <Modal open={open} onClose={handleClose} title="Generate Report" size="sm">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
        <FormField label="Report Type" required>
          <Select
            registration={register("report_type")}
            options={REPORT_TYPES}
          />
        </FormField>
        <FormField label="Format" required>
          <Select
            registration={register("format")}
            options={REPORT_FORMATS}
          />
        </FormField>
        <div className="grid grid-cols-2 gap-3">
          <FormField label="From">
            <input
              type="date"
              {...register("date_from")}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </FormField>
          <FormField label="To">
            <input
              type="date"
              {...register("date_to")}
              className="w-full rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
            />
          </FormField>
        </div>
        <p className="text-xs text-gray-500">
          The file will be downloaded automatically once it is ready.
        </p>
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" loading={loading}>
            Download
          </Button>
        </div>
      </form>
    </Modal>
  );
}
