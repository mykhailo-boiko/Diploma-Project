"use client";

import { useState, useMemo } from "react";
import { Card } from "@/components/ui";
import Button from "@/components/ui/Button";
import StatusBadge from "@/components/ui/StatusBadge";
import { Modal } from "@/components/ui/Modal";
import { FormField, Select } from "@/components/ui/FormField";
import { useForm } from "react-hook-form";
import { toastSuccess, toastError } from "@/lib/toast";
import {
  useSalesSummary,
  useSalesTrends,
  useInventorySummary,
  useLogisticsPerformance,
  useAnomalies,
  useOptimizations,
  useGenerateReport,
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
  const generateReport = useGenerateReport();

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
            logistics ? `${logistics.on_time_rate.toFixed(1)}%` : "-"
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
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900">
            <AlertTriangle className="h-5 w-5 text-yellow-500" />
            Anomaly Alerts
            {anomalyList.length > 0 && (
              <span className="rounded-full bg-red-100 px-2 py-0.5 text-xs font-medium text-red-700">
                {anomalyList.length}
              </span>
            )}
          </h2>
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
          <h2 className="mb-4 flex items-center gap-2 text-lg font-semibold text-gray-900">
            <Lightbulb className="h-5 w-5 text-amber-500" />
            Reorder Recommendations
          </h2>
          {optimizationList.length > 0 ? (
            <div className="overflow-x-auto">
              <table className="min-w-full text-sm">
                <thead>
                  <tr className="border-b text-left text-gray-500">
                    <th className="pb-2 pr-4 font-medium">Product</th>
                    <th className="pb-2 pr-4 font-medium">SKU</th>
                    <th className="pb-2 pr-4 text-right font-medium">Stock</th>
                    <th className="pb-2 pr-4 text-right font-medium">Reorder Pt</th>
                    <th className="pb-2 pr-4 text-right font-medium">Recommended</th>
                    <th className="pb-2 font-medium">Urgency</th>
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
            <EmptyState message="No reorder recommendations at this time" />
          )}
        </div>
      </Card>

      {}
      <ReportModal
        open={reportModalOpen}
        onClose={() => setReportModalOpen(false)}
        onSubmit={(values) => {
          generateReport.mutate(values, {
            onSuccess: () => {
              toastSuccess("Report generated successfully");
              setReportModalOpen(false);
            },
            onError: toastError,
          });
        }}
        loading={generateReport.isPending}
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
          <span className="font-medium">{data.total_quantity.toLocaleString()}</span>
        </div>
        <div>
          <span className="text-gray-500">Low Stock:</span>{" "}
          <span className="font-medium text-red-600">{data.low_stock_count}</span>
        </div>
        <div>
          <span className="text-gray-500">Turnover:</span>{" "}
          <span className="font-medium">{data.turnover_rate.toFixed(2)}</span>
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
            {data.on_time_rate.toFixed(1)}%
          </span>
        </div>
        <div>
          <span className="text-gray-500">Avg Delivery:</span>{" "}
          <span className="font-medium">{data.avg_delivery_h.toFixed(1)}h</span>
        </div>
      </div>
    </div>
  );
}

function AnomalyCard({ anomaly }: { anomaly: Anomaly }) {
  return (
    <div
      className={`rounded-lg border p-3 ${
        anomaly.severity === "critical"
          ? "border-red-200 bg-red-50"
          : "border-yellow-200 bg-yellow-50"
      }`}
    >
      <div className="mb-1 flex items-center justify-between">
        <span className="text-sm font-medium text-gray-900">{anomaly.metric}</span>
        <StatusBadge
          status={anomaly.severity === "critical" ? "failed" : "pending"}
        />
      </div>
      <p className="text-sm text-gray-600">{anomaly.message}</p>
      <div className="mt-1 flex gap-4 text-xs text-gray-400">
        <span>Value: {anomaly.value.toFixed(2)}</span>
        <span>Threshold: {anomaly.threshold.toFixed(2)}</span>
        <span>{anomaly.date}</span>
      </div>
    </div>
  );
}

function OptimizationRow({ opt }: { opt: Optimization }) {
  return (
    <tr className="border-b last:border-0">
      <td className="py-2 pr-4 font-medium text-gray-900">{opt.product_name}</td>
      <td className="py-2 pr-4 text-gray-500">{opt.product_sku}</td>
      <td className="py-2 pr-4 text-right">{opt.current_stock}</td>
      <td className="py-2 pr-4 text-right">{opt.reorder_point.toFixed(0)}</td>
      <td className="py-2 pr-4 text-right font-medium">{opt.recommended_order.toFixed(0)}</td>
      <td className="py-2">
        <StatusBadge
          status={
            opt.urgency === "critical"
              ? "failed"
              : opt.urgency === "warning"
                ? "pending"
                : "active"
          }
        />
      </td>
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
        <div className="flex justify-end gap-3 pt-2">
          <Button variant="secondary" type="button" onClick={handleClose}>
            Cancel
          </Button>
          <Button type="submit" loading={loading}>
            Generate
          </Button>
        </div>
      </form>
    </Modal>
  );
}
