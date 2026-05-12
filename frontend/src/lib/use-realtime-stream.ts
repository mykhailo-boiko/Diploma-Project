"use client";

import { useQueryClient } from "@tanstack/react-query";
import { useEffect, useRef, useState } from "react";

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type RealtimeEvent = {
  type: string;
  source: string;
  timestamp: string;
  subject: string;
  data?: Record<string, unknown> | null;
};

export type RealtimeStatus = "idle" | "connecting" | "open" | "error";

type Listener = (ev: RealtimeEvent) => void;

class RealtimeBus {
  private listeners = new Set<Listener>();

  subscribe(fn: Listener) {
    this.listeners.add(fn);
    return () => {
      this.listeners.delete(fn);
    };
  }

  emit(ev: RealtimeEvent) {
    this.listeners.forEach((fn) => {
      try {
        fn(ev);
      } catch {
      }
    });
  }
}

let bus: RealtimeBus | null = null;
let connection: {
  controller: AbortController;
  status: RealtimeStatus;
  retry: number;
} | null = null;
let statusListeners = new Set<(s: RealtimeStatus) => void>();
let currentStatus: RealtimeStatus = "idle";

function setStatus(s: RealtimeStatus) {
  currentStatus = s;
  statusListeners.forEach((fn) => fn(s));
}

function getBus(): RealtimeBus {
  if (!bus) bus = new RealtimeBus();
  return bus;
}

async function streamLoop() {
  if (!connection) return;
  const ctrl = connection.controller;
  while (!ctrl.signal.aborted) {
    setStatus("connecting");
    const token = typeof window !== "undefined" ? localStorage.getItem("access_token") : null;
    if (!token) {
      await new Promise((r) => setTimeout(r, 1000));
      continue;
    }
    try {
      const res = await fetch(`${API_BASE}/api/v1/events/stream`, {
        headers: { Authorization: `Bearer ${token}`, Accept: "text/event-stream" },
        signal: ctrl.signal,
      });
      if (!res.ok || !res.body) {
        throw new Error(`stream HTTP ${res.status}`);
      }
      setStatus("open");
      connection.retry = 0;
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      while (true) {
        const { value, done } = await reader.read();
        if (done) break;
        buffer += decoder.decode(value, { stream: true });
        let idx;
        while ((idx = buffer.indexOf("\n\n")) !== -1) {
          const chunk = buffer.slice(0, idx);
          buffer = buffer.slice(idx + 2);
          handleChunk(chunk);
        }
      }
    } catch (err) {
      if (ctrl.signal.aborted) return;
      setStatus("error");
    }
    if (ctrl.signal.aborted) return;
    const backoff = Math.min(15000, 800 * Math.pow(2, connection.retry));
    connection.retry += 1;
    await new Promise((r) => setTimeout(r, backoff));
  }
}

function handleChunk(chunk: string) {
  const lines = chunk.split("\n");
  let eventType = "";
  let dataRaw = "";
  for (const line of lines) {
    if (line.startsWith("event:")) eventType = line.slice(6).trim();
    else if (line.startsWith("data:")) dataRaw += line.slice(5).trim();
  }
  if (!dataRaw) return;
  try {
    const ev = JSON.parse(dataRaw) as RealtimeEvent;
    if (!ev.type && eventType) ev.type = eventType;
    getBus().emit(ev);
  } catch {
  }
}

export function ensureRealtimeConnected() {
  if (typeof window === "undefined") return;
  if (connection && !connection.controller.signal.aborted) return;
  const controller = new AbortController();
  connection = { controller, status: "idle", retry: 0 };
  streamLoop();
}

export function disconnectRealtime() {
  if (connection) {
    connection.controller.abort();
    connection = null;
  }
  setStatus("idle");
}

export function useRealtimeStream() {
  const qc = useQueryClient();
  const [status, setLocalStatus] = useState<RealtimeStatus>(currentStatus);

  useEffect(() => {
    statusListeners.add(setLocalStatus);
    ensureRealtimeConnected();
    return () => {
      statusListeners.delete(setLocalStatus);
    };
  }, []);

  const handlerRef = useRef<Listener | null>(null);
  useEffect(() => {
    if (handlerRef.current) return;
    const handler: Listener = (ev) => {
      switch (ev.type) {
        case "order.created":
        case "order.updated":
        case "order.cancelled":
          qc.invalidateQueries({ queryKey: ["orders"] });
          qc.invalidateQueries({ queryKey: ["analytics", "sales"] });
          qc.invalidateQueries({ queryKey: ["dashboard"] });
          break;
        case "shipment.created":
        case "shipment.updated":
        case "shipment.out_for_delivery":
        case "shipment.delivered":
        case "shipment.attempted":
        case "shipment.returned":
        case "shipment.redirected":
          qc.invalidateQueries({ queryKey: ["shipments"] });
          qc.invalidateQueries({ queryKey: ["dashboard"] });
          qc.invalidateQueries({ queryKey: ["analytics", "logistics"] });
          break;
        case "stock.changed":
        case "stock.low":
          qc.invalidateQueries({ queryKey: ["stock"] });
          qc.invalidateQueries({ queryKey: ["products"] });
          qc.invalidateQueries({ queryKey: ["dashboard"] });
          break;
        case "notification.new":
          qc.invalidateQueries({ queryKey: ["notifications"] });
          break;
        case "analytics.updated":
          qc.invalidateQueries({ queryKey: ["analytics"] });
          qc.invalidateQueries({ queryKey: ["dashboard"] });
          break;
      }
    };
    handlerRef.current = handler;
    const unsub = getBus().subscribe(handler);
    return () => {
      unsub();
      handlerRef.current = null;
    };
  }, [qc]);

  return { status };
}

export function useRealtimeEvents(maxBuffer = 50) {
  const [events, setEvents] = useState<RealtimeEvent[]>([]);
  useEffect(() => {
    const unsub = getBus().subscribe((ev) => {
      if (ev.type === "connection.established") return;
      setEvents((prev) => [ev, ...prev].slice(0, maxBuffer));
    });
    return unsub;
  }, [maxBuffer]);
  return events;
}

export function getRealtimeStatus() {
  return currentStatus;
}
