"use client";

import { useApiMutation, useApiQuery, type SingleResponse } from "./api-hooks";

export type SimulatorScenario =
  | "idle"
  | "steady"
  | "holiday_spike"
  | "carrier_failure"
  | "demand_surge";

export interface SimulatorStatus {
  enabled: boolean;
  scenario: SimulatorScenario;
  speed: number;
  started_at: string;
  uptime_secs: number;
  counters: {
    orders_created: number;
    orders_progressed: number;
    orders_cancelled: number;
    shipments_advanced: number;
    shipment_events: number;
    stock_adjustments: number;
    notifications_sent: number;
    errors: number;
  };
}

export function useSimulatorStatus() {
  return useApiQuery<SingleResponse<SimulatorStatus>>(
    ["simulator", "status"],
    "/api/v1/simulator/status",
    { refetchInterval: 2000 },
  );
}

export function useStartSimulator() {
  return useApiMutation<SingleResponse<SimulatorStatus>, { scenario: SimulatorScenario; speed: number }>(
    "POST",
    () => "/api/v1/simulator/start",
    { invalidateKeys: [["simulator", "status"]] },
  );
}

export function useStopSimulator() {
  return useApiMutation<SingleResponse<SimulatorStatus>, Record<string, never>>(
    "POST",
    () => "/api/v1/simulator/stop",
    { invalidateKeys: [["simulator", "status"]] },
  );
}

export function useSetSimulatorSpeed() {
  return useApiMutation<SingleResponse<SimulatorStatus>, { speed: number }>(
    "POST",
    () => "/api/v1/simulator/speed",
    { invalidateKeys: [["simulator", "status"]] },
  );
}

export function useSetSimulatorScenario() {
  return useApiMutation<SingleResponse<SimulatorStatus>, { scenario: SimulatorScenario }>(
    "POST",
    () => "/api/v1/simulator/scenario",
    { invalidateKeys: [["simulator", "status"]] },
  );
}
