"use client";

import { useState } from "react";
import {
  useSimulatorStatus,
  useStartSimulator,
  useStopSimulator,
  useSetSimulatorSpeed,
  useSetSimulatorScenario,
  type SimulatorScenario,
} from "@/lib/use-simulator";
import { useAuthStore } from "@/stores/auth";

const scenarios: { value: SimulatorScenario; label: string; description: string }[] = [
  { value: "idle", label: "Idle", description: "All actors paused. Use to stop without flipping enabled flag." },
  { value: "steady", label: "Steady", description: "Baseline traffic: ~1 order/min, ~6 shipment ticks/min, light stock churn." },
  { value: "holiday_spike", label: "Holiday spike", description: "3× order rate, faster shipment progression, heavier stock motion." },
  { value: "carrier_failure", label: "Carrier failure", description: "Same baseline traffic but 60% of out-for-delivery attempts fail." },
  { value: "demand_surge", label: "Demand surge", description: "5× orders, aggressive shipment + inventory churn — for stress demos." },
];

const speedOptions = [1, 2, 5, 10, 25, 50];

export default function SimulatorAdminPage() {
  const user = useAuthStore((s) => s.user);
  const status = useSimulatorStatus();
  const start = useStartSimulator();
  const stop = useStopSimulator();
  const setSpeed = useSetSimulatorSpeed();
  const setScenario = useSetSimulatorScenario();

  const [pendingScenario, setPendingScenario] = useState<SimulatorScenario | null>(null);
  const [pendingSpeed, setPendingSpeed] = useState<number | null>(null);

  if (user?.role !== "admin") {
    return (
      <div className="rounded-md border border-amber-200 bg-amber-50 p-4 text-sm text-amber-800">
        Admin role required to access the simulator controls.
      </div>
    );
  }

  const data = status.data?.data;
  const enabled = data?.enabled ?? false;
  const scenario = pendingScenario ?? data?.scenario ?? "steady";
  const speed = pendingSpeed ?? data?.speed ?? 1;

  function handleStart() {
    start.mutate({ scenario, speed });
  }

  function handleStop() {
    stop.mutate({});
  }

  function handleScenarioChange(next: SimulatorScenario) {
    setPendingScenario(next);
    if (enabled) {
      setScenario.mutate({ scenario: next });
    }
  }

  function handleSpeedChange(next: number) {
    setPendingSpeed(next);
    if (enabled) {
      setSpeed.mutate({ speed: next });
    }
  }

  return (
    <div className="space-y-6">
      <header className="space-y-1">
        <h1 className="text-2xl font-semibold text-gray-900">Live simulator</h1>
        <p className="text-sm text-gray-600">
          Continuously generates realistic supply-chain traffic so dashboards, charts and the activity feed update in real time.
        </p>
      </header>

      <section className="grid gap-4 md:grid-cols-3">
        <StatusCard
          title="Status"
          value={enabled ? "Running" : "Stopped"}
          tone={enabled ? "positive" : "neutral"}
        />
        <StatusCard
          title="Scenario"
          value={scenario}
          tone="info"
        />
        <StatusCard
          title="Speed"
          value={`${speed.toFixed(speed % 1 === 0 ? 0 : 1)}×`}
          tone="accent"
        />
      </section>

      <section className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
        <div className="flex flex-wrap items-center gap-3">
          {enabled ? (
            <button
              type="button"
              onClick={handleStop}
              disabled={stop.isPending}
              className="rounded-md bg-red-600 px-4 py-2 text-sm font-semibold text-white hover:bg-red-700 disabled:opacity-60"
            >
              {stop.isPending ? "Stopping…" : "Stop simulator"}
            </button>
          ) : (
            <button
              type="button"
              onClick={handleStart}
              disabled={start.isPending}
              className="rounded-md bg-emerald-600 px-4 py-2 text-sm font-semibold text-white hover:bg-emerald-700 disabled:opacity-60"
            >
              {start.isPending ? "Starting…" : "Start simulator"}
            </button>
          )}
          {enabled && data && (
            <span className="text-xs text-gray-500">
              Uptime: {formatUptime(data.uptime_secs)} · started {new Date(data.started_at).toLocaleString()}
            </span>
          )}
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold text-gray-700">Scenario</h2>
        <div className="grid gap-3 md:grid-cols-2 xl:grid-cols-3">
          {scenarios.map((s) => (
            <button
              key={s.value}
              type="button"
              onClick={() => handleScenarioChange(s.value)}
              className={`rounded-md border p-3 text-left transition ${
                scenario === s.value
                  ? "border-blue-500 bg-blue-50 ring-1 ring-blue-300"
                  : "border-gray-200 bg-white hover:border-gray-300"
              }`}
            >
              <div className="text-sm font-semibold text-gray-900">{s.label}</div>
              <div className="mt-1 text-xs text-gray-600">{s.description}</div>
            </button>
          ))}
        </div>
      </section>

      <section className="space-y-3">
        <h2 className="text-sm font-semibold text-gray-700">Speed multiplier</h2>
        <div className="flex flex-wrap gap-2">
          {speedOptions.map((v) => (
            <button
              key={v}
              type="button"
              onClick={() => handleSpeedChange(v)}
              className={`rounded-md border px-3 py-1.5 text-sm transition ${
                speed === v
                  ? "border-blue-500 bg-blue-50 text-blue-700"
                  : "border-gray-200 bg-white text-gray-700 hover:border-gray-300"
              }`}
            >
              {v}×
            </button>
          ))}
        </div>
        <p className="text-xs text-gray-500">
          Higher multipliers shrink per-actor intervals and accelerate the rolling shipment lifecycle. 1× ≈ real-world cadence.
        </p>
      </section>

      <section className="rounded-lg border border-gray-200 bg-white p-5 shadow-sm">
        <h2 className="mb-3 text-sm font-semibold text-gray-700">Counters (since start)</h2>
        {data ? (
          <div className="grid gap-3 sm:grid-cols-2 md:grid-cols-4">
            <Counter label="Orders created" value={data.counters.orders_created} />
            <Counter label="Orders progressed" value={data.counters.orders_progressed} />
            <Counter label="Orders cancelled" value={data.counters.orders_cancelled} />
            <Counter label="Shipments advanced" value={data.counters.shipments_advanced} />
            <Counter label="Shipment events" value={data.counters.shipment_events} />
            <Counter label="Stock adjustments" value={data.counters.stock_adjustments} />
            <Counter label="Notifications" value={data.counters.notifications_sent} />
            <Counter label="Errors" value={data.counters.errors} tone={data.counters.errors > 0 ? "warning" : "neutral"} />
          </div>
        ) : (
          <div className="text-xs text-gray-500">Loading…</div>
        )}
      </section>
    </div>
  );
}

function StatusCard({ title, value, tone }: { title: string; value: string; tone: "positive" | "neutral" | "info" | "accent" }) {
  const toneClass: Record<string, string> = {
    positive: "border-emerald-200 bg-emerald-50 text-emerald-900",
    neutral: "border-gray-200 bg-white text-gray-900",
    info: "border-blue-200 bg-blue-50 text-blue-900",
    accent: "border-purple-200 bg-purple-50 text-purple-900",
  };
  return (
    <div className={`rounded-lg border p-4 shadow-sm ${toneClass[tone]}`}>
      <div className="text-xs font-medium uppercase tracking-wide opacity-70">{title}</div>
      <div className="mt-1 text-xl font-semibold capitalize">{value.replace(/_/g, " ")}</div>
    </div>
  );
}

function Counter({ label, value, tone = "neutral" }: { label: string; value: number; tone?: "neutral" | "warning" }) {
  return (
    <div className={`rounded-md border p-3 ${tone === "warning" ? "border-amber-200 bg-amber-50" : "border-gray-100 bg-gray-50"}`}>
      <div className="text-[11px] uppercase tracking-wide text-gray-500">{label}</div>
      <div className="mt-1 text-lg font-semibold text-gray-900">{value.toLocaleString()}</div>
    </div>
  );
}

function formatUptime(secs: number): string {
  if (secs < 60) return `${secs}s`;
  if (secs < 3600) return `${Math.floor(secs / 60)}m ${secs % 60}s`;
  const h = Math.floor(secs / 3600);
  const m = Math.floor((secs % 3600) / 60);
  return `${h}h ${m}m`;
}
