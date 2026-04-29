"use client";

import { useState, useCallback } from "react";
import { Settings } from "lucide-react";
import DataTable, { type Column } from "@/components/ui/DataTable";
import StatusBadge from "@/components/ui/StatusBadge";
import Button from "@/components/ui/Button";
import { Modal } from "@/components/ui/Modal";
import { toastSuccess, toastError } from "@/lib/toast";
import {
  useNotifications,
  useUnreadCount,
  useMarkAsRead,
  useNotificationPreferences,
  useUpdatePreference,
  type Notification,
  type NotificationPreference,
} from "@/lib/use-notifications";

const NOTIFICATION_TYPES = [
  { value: "", label: "All types" },
  { value: "order_created", label: "Order Created" },
  { value: "order_updated", label: "Order Updated" },
  { value: "order_cancelled", label: "Order Cancelled" },
  { value: "low_stock", label: "Low Stock" },
  { value: "stock_changed", label: "Stock Changed" },
  { value: "shipment_created", label: "Shipment Created" },
  { value: "shipment_updated", label: "Shipment Updated" },
  { value: "system", label: "System" },
];

const columns: Column<Notification>[] = [
  {
    key: "status",
    header: "",
    render: (row) =>
      row.status === "pending" ? (
        <span className="inline-block h-2 w-2 rounded-full bg-blue-500" title="Unread" />
      ) : null,
    className: "w-8",
  },
  {
    key: "type",
    header: "Type",
    render: (row) => (
      <StatusBadge status={row.type} />
    ),
  },
  {
    key: "title",
    header: "Title",
    render: (row) => (
      <span className={row.status === "pending" ? "font-semibold" : ""}>
        {row.title}
      </span>
    ),
  },
  {
    key: "message",
    header: "Message",
    render: (row) => (
      <span className="line-clamp-1 text-gray-500">{row.message}</span>
    ),
  },
  {
    key: "created_at",
    header: "Date",
    render: (row) => {
      const d = new Date(row.created_at);
      return d.toLocaleDateString() + " " + d.toLocaleTimeString([], { hour: "2-digit", minute: "2-digit" });
    },
  },
];

export default function NotificationsPage() {
  const [page, setPage] = useState(1);
  const [typeFilter, setTypeFilter] = useState("");
  const [prefsOpen, setPrefsOpen] = useState(false);
  const pageSize = 10;

  const { data, isLoading } = useNotifications({
    page,
    pageSize,
    type: typeFilter || undefined,
  });
  const unreadCount = useUnreadCount();
  const markAsRead = useMarkAsRead();

  const notifications = data?.data ?? [];
  const total = data?.meta?.total ?? 0;
  const unread = unreadCount.data?.data?.unread_count ?? 0;

  const handleMarkRead = useCallback(
    (row: Notification) => {
      if (row.status === "read") return;
      markAsRead.mutate(
        { id: row.id },
        {
          onSuccess: () => toastSuccess("Marked as read"),
          onError: toastError,
        },
      );
    },
    [markAsRead],
  );

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-3">
          <h1 className="text-2xl font-bold text-gray-900">Notifications</h1>
          {unread > 0 && (
            <span className="rounded-full bg-blue-100 px-2.5 py-0.5 text-sm font-medium text-blue-700">
              {unread} unread
            </span>
          )}
        </div>
        <Button variant="secondary" onClick={() => setPrefsOpen(true)}>
          <Settings className="mr-1 h-4 w-4" />
          Preferences
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
          {NOTIFICATION_TYPES.map((t) => (
            <option key={t.value} value={t.value}>
              {t.label}
            </option>
          ))}
        </select>
      </div>

      {}
      <DataTable<Notification>
        columns={columns}
        data={notifications}
        total={total}
        page={page}
        pageSize={pageSize}
        onPageChange={setPage}
        loading={isLoading}
        emptyMessage="No notifications"
        rowKey={(row) => row.id}
        onRowClick={handleMarkRead}
      />

      {}
      <PreferencesModal
        open={prefsOpen}
        onClose={() => setPrefsOpen(false)}
      />
    </div>
  );
}

function PreferencesModal({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const prefs = useNotificationPreferences();
  const updatePref = useUpdatePreference();
  const preferences = prefs.data?.data ?? [];

  const notifTypes = [
    "order_created",
    "order_updated",
    "order_cancelled",
    "low_stock",
    "stock_changed",
    "shipment_created",
    "shipment_updated",
    "system",
  ];

  function getPref(type: string): NotificationPreference | undefined {
    return preferences.find((p) => p.type === type);
  }

  function togglePref(type: string, channel: "in_app" | "email" | "sms") {
    const existing = getPref(type);
    const current = {
      in_app: existing?.in_app ?? true,
      email: existing?.email ?? true,
      sms: existing?.sms ?? false,
    };
    updatePref.mutate(
      { type, ...current, [channel]: !current[channel] },
      { onError: toastError },
    );
  }

  function formatType(type: string): string {
    return type
      .split("_")
      .map((w) => w.charAt(0).toUpperCase() + w.slice(1))
      .join(" ");
  }

  return (
    <Modal open={open} onClose={onClose} title="Notification Preferences" size="md">
      <div className="overflow-x-auto">
        <table className="min-w-full text-sm">
          <thead>
            <tr className="border-b text-left text-gray-500">
              <th className="pb-2 pr-6 font-medium">Type</th>
              <th className="pb-2 px-4 text-center font-medium">In-App</th>
              <th className="pb-2 px-4 text-center font-medium">Email</th>
              <th className="pb-2 px-4 text-center font-medium">SMS</th>
            </tr>
          </thead>
          <tbody>
            {notifTypes.map((type) => {
              const pref = getPref(type);
              const inApp = pref?.in_app ?? true;
              const email = pref?.email ?? true;
              const sms = pref?.sms ?? false;
              return (
                <tr key={type} className="border-b last:border-0">
                  <td className="py-2 pr-6 font-medium text-gray-900">
                    {formatType(type)}
                  </td>
                  <td className="py-2 px-4 text-center">
                    <ToggleButton
                      enabled={inApp}
                      onClick={() => togglePref(type, "in_app")}
                    />
                  </td>
                  <td className="py-2 px-4 text-center">
                    <ToggleButton
                      enabled={email}
                      onClick={() => togglePref(type, "email")}
                    />
                  </td>
                  <td className="py-2 px-4 text-center">
                    <ToggleButton
                      enabled={sms}
                      onClick={() => togglePref(type, "sms")}
                    />
                  </td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
      <div className="mt-4 flex justify-end">
        <Button variant="secondary" onClick={onClose}>
          Close
        </Button>
      </div>
    </Modal>
  );
}

function ToggleButton({
  enabled,
  onClick,
}: {
  enabled: boolean;
  onClick: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`inline-flex h-6 w-10 items-center rounded-full transition-colors ${
        enabled ? "bg-blue-600" : "bg-gray-200"
      }`}
    >
      <span
        className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
          enabled ? "translate-x-5" : "translate-x-1"
        }`}
      />
    </button>
  );
}
