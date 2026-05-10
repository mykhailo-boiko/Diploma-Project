"use client";

import { useAuthStore } from "@/stores/auth";
import { formatRole } from "@/lib/roles";
import { StatCard, Card, StatusBadge } from "@/components/ui";
import {
  ShoppingCart,
  Package,
  Truck,
  Bell,
  DollarSign,
  AlertTriangle,
  TrendingUp,
  BarChart3,
  Warehouse,
  CheckCircle,
  Clock,
  XCircle,
} from "lucide-react";
import {
  useOrderStats,
  useUnreadCount,
  useLowStock,
  useLogisticsPerformance,
  useSalesSummary,
  useSalesTrends,
  useInventorySummary,
  useAnalyticsLogisticsPerf,
  useAnomalies,
  useRecentOrders,
  canSee,
} from "@/lib/use-dashboard";
import type { Role } from "@/lib/types";
import type { SalesTrend, Anomaly, Order, LowStockItem } from "@/lib/use-dashboard";
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
import { useRouter } from "next/navigation";
import { safeFixed } from "@/lib/format";

const PIE_COLORS = ["#3b82f6", "#10b981", "#f59e0b", "#ef4444", "#8b5cf6", "#ec4899", "#6366f1"];

function fmtCurrency(n: number) {
  return new Intl.NumberFormat("en-US", {
    style: "currency",
    currency: "USD",
    maximumFractionDigits: 0,
  }).format(n);
}

export default function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  if (!user) return null;

  const role = user.role;

  return (
    <div>
      <h1 className="text-2xl font-bold text-gray-900">Dashboard</h1>
      <p className="mt-1 text-gray-600">
        Welcome back, {user.first_name}! You are logged in as{" "}
        {formatRole(user.role)}.
      </p>

      <div className="mt-6 space-y-6">
        <KPIRow role={role} />

        {canSee(role, "orders") && <OrdersSection />}
        {canSee(role, "analytics") && <AnalyticsSection />}
        {canSee(role, "inventory") && <InventorySection />}
        {canSee(role, "logistics") && <LogisticsSection />}
      </div>
    </div>
  );
}


function KPIRow({ role }: { role: Role }) {
  const orderStats = useOrderStats(canSee(role, "orders"));
  const unread = useUnreadCount();
  const salesSummary = useSalesSummary(canSee(role, "analytics") || canSee(role, "orders"));
  const inventorySummary = useInventorySummary(canSee(role, "inventory"));
  const logisticsPerf = useLogisticsPerformance(canSee(role, "logistics"));

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {canSee(role, "orders") && (
        <StatCard
          title="Total Orders"
          value={orderStats.data?.data.total_orders ?? "—"}
          subtitle="All time"
          icon={ShoppingCart}
        />
      )}

      {(canSee(role, "analytics") || canSee(role, "orders")) && (
        <StatCard
          title="Revenue (30d)"
          value={
            salesSummary.data
              ? fmtCurrency(salesSummary.data.data.total_revenue)
              : "—"
          }
          subtitle={
            salesSummary.data
              ? `Avg order: ${fmtCurrency(salesSummary.data.data.avg_order_value)}`
              : undefined
          }
          icon={DollarSign}
        />
      )}

      {canSee(role, "inventory") && (
        <StatCard
          title="Stock Available"
          value={inventorySummary.data?.data.total_available ?? "—"}
          subtitle={
            inventorySummary.data
              ? `${inventorySummary.data.data.low_stock_count} low stock`
              : undefined
          }
          icon={Package}
        />
      )}

      {canSee(role, "logistics") && (
        <StatCard
          title="On-Time Rate"
          value={
            typeof logisticsPerf.data?.data?.on_time_rate === "number"
              ? `${safeFixed(logisticsPerf.data.data.on_time_rate, 1)}%`
              : "—"
          }
          subtitle={
            typeof logisticsPerf.data?.data?.late === "number"
              ? `${logisticsPerf.data.data.late} late deliveries`
              : undefined
          }
          icon={Truck}
        />
      )}

      <StatCard
        title="Notifications"
        value={unread.data?.data.unread_count ?? "—"}
        subtitle="Unread"
        icon={Bell}
      />
    </div>
  );
}


function OrdersSection() {
  const orderStats = useOrderStats(true);
  const recentOrders = useRecentOrders(true);
  const router = useRouter();

  const byStatus = orderStats.data?.data.by_status ?? [];
  const pieData = byStatus.map((s) => ({ name: s.status, value: s.count }));

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Orders by Status
        </h3>
        {pieData.length > 0 ? (
          <ResponsiveContainer width="100%" height={260}>
            <PieChart>
              <Pie
                data={pieData}
                cx="50%"
                cy="50%"
                innerRadius={60}
                outerRadius={100}
                paddingAngle={2}
                dataKey="value"
                label={({ name, value }) => `${name}: ${value}`}
              >
                {pieData.map((_, i) => (
                  <Cell key={i} fill={PIE_COLORS[i % PIE_COLORS.length]} />
                ))}
              </Pie>
              <Tooltip />
            </PieChart>
          </ResponsiveContainer>
        ) : (
          <EmptyState message="No order data available" />
        )}
      </Card>

      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Recent Orders
        </h3>
        {recentOrders.data?.data && recentOrders.data.data.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="border-b text-left text-gray-500">
                  <th className="pb-2 font-medium">Customer</th>
                  <th className="pb-2 font-medium">Amount</th>
                  <th className="pb-2 font-medium">Status</th>
                </tr>
              </thead>
              <tbody>
                {recentOrders.data.data.map((order: Order) => (
                  <tr
                    key={order.id}
                    className="cursor-pointer border-b last:border-0 hover:bg-gray-50"
                    onClick={() => router.push(`/orders/${order.id}`)}
                  >
                    <td className="py-2">{order.customer_name}</td>
                    <td className="py-2">{fmtCurrency(order.total_amount)}</td>
                    <td className="py-2">
                      <StatusBadge status={order.status} />
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <EmptyState message="No recent orders" />
        )}
        <QuickAction label="View all orders" href="/orders" />
      </Card>
    </div>
  );
}


function AnalyticsSection() {
  const trends = useSalesTrends(true);
  const anomalies = useAnomalies(true);

  const trendData = (trends.data?.data ?? []).map((t: SalesTrend) => ({
    period: t.period,
    revenue: Math.round(t.total_revenue),
    orders: t.total_orders,
  }));

  const anomalyList = anomalies.data?.data ?? [];

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Revenue Trend (30d)
        </h3>
        {trendData.length > 0 ? (
          <ResponsiveContainer width="100%" height={260}>
            <LineChart data={trendData}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="period" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip formatter={(v) => fmtCurrency(Number(v))} />
              <Legend />
              <Line
                type="monotone"
                dataKey="revenue"
                stroke="#3b82f6"
                strokeWidth={2}
                dot={false}
                name="Revenue"
              />
            </LineChart>
          </ResponsiveContainer>
        ) : (
          <EmptyState message="No trend data available" />
        )}
      </Card>

      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Anomaly Alerts
        </h3>
        {anomalyList.length > 0 ? (
          <div className="space-y-3 max-h-[260px] overflow-y-auto">
            {anomalyList.map((a: Anomaly, i: number) => (
              <div
                key={i}
                className={`rounded-lg border p-3 ${
                  a.severity === "critical"
                    ? "border-red-200 bg-red-50"
                    : "border-yellow-200 bg-yellow-50"
                }`}
              >
                <div className="flex items-center gap-2">
                  <AlertTriangle
                    className={`h-4 w-4 ${
                      a.severity === "critical"
                        ? "text-red-500"
                        : "text-yellow-500"
                    }`}
                  />
                  <span className="text-sm font-medium">{a.type}</span>
                  <span
                    className={`ml-auto rounded-full px-2 py-0.5 text-xs font-medium ${
                      a.severity === "critical"
                        ? "bg-red-100 text-red-700"
                        : "bg-yellow-100 text-yellow-700"
                    }`}
                  >
                    {a.severity}
                  </span>
                </div>
                <p className="mt-1 text-sm text-gray-600">{a.message}</p>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex h-[200px] items-center justify-center text-sm text-gray-400">
            <CheckCircle className="mr-2 h-5 w-5 text-green-400" />
            No anomalies detected
          </div>
        )}
        <QuickAction label="View analytics" href="/analytics" />
      </Card>
    </div>
  );
}


function InventorySection() {
  const inventorySummary = useInventorySummary(true);
  const lowStock = useLowStock(true);

  const summary = inventorySummary.data?.data;
  const lowItems = lowStock.data?.data ?? [];

  const stockBreakdown = summary
    ? [
        { name: "Available", value: summary.total_available },
        { name: "Reserved", value: summary.total_reserved },
      ]
    : [];

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Stock Overview
        </h3>
        {summary ? (
          <div className="space-y-4">
            <MiniStat
              icon={Package}
              label="Total Products"
              value={summary.total_products}
            />
            <MiniStat
              icon={Warehouse}
              label="Total Quantity"
              value={(summary.total_quantity ?? 0).toLocaleString()}
            />
            <MiniStat
              icon={TrendingUp}
              label="Turnover Rate"
              value={
                typeof summary.turnover_rate === "number"
                  ? `${safeFixed(summary.turnover_rate, 2)}x`
                  : "—"
              }
            />
            <MiniStat
              icon={AlertTriangle}
              label="Low Stock Items"
              value={summary.low_stock_count ?? 0}
              danger={(summary.low_stock_count ?? 0) > 0}
            />
          </div>
        ) : (
          <EmptyState message="No inventory data" />
        )}
      </Card>

      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Stock Breakdown
        </h3>
        {stockBreakdown.length > 0 ? (
          <ResponsiveContainer width="100%" height={220}>
            <BarChart data={stockBreakdown}>
              <CartesianGrid strokeDasharray="3 3" />
              <XAxis dataKey="name" tick={{ fontSize: 12 }} />
              <YAxis tick={{ fontSize: 12 }} />
              <Tooltip />
              <Bar dataKey="value" radius={[4, 4, 0, 0]}>
                <Cell fill="#3b82f6" />
                <Cell fill="#f59e0b" />
              </Bar>
            </BarChart>
          </ResponsiveContainer>
        ) : (
          <EmptyState message="No stock data" />
        )}
      </Card>

      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Low Stock Alerts
        </h3>
        {lowItems.length > 0 ? (
          <div className="space-y-2 max-h-[220px] overflow-y-auto">
            {lowItems.map((item: LowStockItem) => (
              <div
                key={item.id}
                className="flex items-center justify-between rounded border border-red-100 bg-red-50 px-3 py-2 text-sm"
              >
                <div>
                  <span className="font-medium">{item.product_name}</span>
                  <span className="ml-2 text-gray-500">{item.product_sku}</span>
                </div>
                <span className="font-semibold text-red-600">
                  {item.available}/{item.min_threshold}
                </span>
              </div>
            ))}
          </div>
        ) : (
          <div className="flex h-[200px] items-center justify-center text-sm text-gray-400">
            <CheckCircle className="mr-2 h-5 w-5 text-green-400" />
            All stock levels normal
          </div>
        )}
        <QuickAction label="View stock" href="/stock" />
      </Card>
    </div>
  );
}


function LogisticsSection() {
  const perf = useLogisticsPerformance(true);
  const analyticsPerf = useAnalyticsLogisticsPerf(true);

  const perfData = perf.data?.data;
  const aPerf = analyticsPerf.data?.data;

  const deliveryData = perfData
    ? [
        { name: "On Time", value: perfData.on_time },
        { name: "Late", value: perfData.late },
      ]
    : [];

  return (
    <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Delivery Performance
        </h3>
        {deliveryData.length > 0 && (perfData!.on_time > 0 || perfData!.late > 0) ? (
          <ResponsiveContainer width="100%" height={260}>
            <PieChart>
              <Pie
                data={deliveryData}
                cx="50%"
                cy="50%"
                innerRadius={60}
                outerRadius={100}
                paddingAngle={2}
                dataKey="value"
                label={({ name, value }) => `${name}: ${value}`}
              >
                <Cell fill="#10b981" />
                <Cell fill="#ef4444" />
              </Pie>
              <Tooltip />
              <Legend />
            </PieChart>
          </ResponsiveContainer>
        ) : (
          <EmptyState message="No delivery data yet" />
        )}
      </Card>

      {}
      <Card>
        <h3 className="mb-4 text-lg font-semibold text-gray-900">
          Logistics Metrics
        </h3>
        {aPerf ? (
          <div className="space-y-4">
            <MiniStat
              icon={Truck}
              label="Total Shipments"
              value={aPerf.total_shipments}
            />
            <MiniStat
              icon={CheckCircle}
              label="Delivered"
              value={aPerf.delivered_count}
            />
            <MiniStat
              icon={XCircle}
              label="Failed"
              value={aPerf.failed_count}
              danger={aPerf.failed_count > 0}
            />
            <MiniStat
              icon={Clock}
              label="Avg Delivery Time"
              value={
                typeof aPerf.avg_delivery_h === "number"
                  ? `${safeFixed(aPerf.avg_delivery_h, 1)}h`
                  : "—"
              }
            />
            <MiniStat
              icon={BarChart3}
              label="On-Time Rate"
              value={
                typeof aPerf.on_time_rate === "number"
                  ? `${safeFixed(aPerf.on_time_rate, 1)}%`
                  : "—"
              }
            />
          </div>
        ) : perfData ? (
          <div className="space-y-4">
            <MiniStat
              icon={CheckCircle}
              label="On Time"
              value={perfData.on_time}
            />
            <MiniStat
              icon={XCircle}
              label="Late"
              value={perfData.late}
              danger={perfData.late > 0}
            />
            <MiniStat
              icon={BarChart3}
              label="On-Time Rate"
              value={
                typeof perfData.on_time_rate === "number"
                  ? `${safeFixed(perfData.on_time_rate, 1)}%`
                  : "—"
              }
            />
          </div>
        ) : (
          <EmptyState message="No logistics data" />
        )}
        <QuickAction label="View shipments" href="/shipments" />
      </Card>
    </div>
  );
}


function MiniStat({
  icon: Icon,
  label,
  value,
  danger,
}: {
  icon: React.ComponentType<{ className?: string }>;
  label: string;
  value: string | number;
  danger?: boolean;
}) {
  return (
    <div className="flex items-center gap-3">
      <div
        className={`rounded-lg p-2 ${danger ? "bg-red-50" : "bg-gray-100"}`}
      >
        <Icon
          className={`h-4 w-4 ${danger ? "text-red-500" : "text-gray-600"}`}
        />
      </div>
      <div className="flex-1">
        <p className="text-xs text-gray-500">{label}</p>
        <p
          className={`text-sm font-semibold ${danger ? "text-red-600" : "text-gray-900"}`}
        >
          {value}
        </p>
      </div>
    </div>
  );
}

function EmptyState({ message }: { message: string }) {
  return (
    <div className="flex h-[200px] items-center justify-center text-sm text-gray-400">
      {message}
    </div>
  );
}

function QuickAction({ label, href }: { label: string; href: string }) {
  const router = useRouter();
  return (
    <button
      onClick={() => router.push(href)}
      className="mt-4 w-full rounded-lg border border-gray-200 py-2 text-center text-sm font-medium text-blue-600 hover:bg-blue-50"
    >
      {label} &rarr;
    </button>
  );
}
