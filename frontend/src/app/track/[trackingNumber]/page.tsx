"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { Package, Search, ShieldCheck, Truck, Loader2 } from "lucide-react";
import TrackingTimeline, { type ShipmentEvent } from "@/components/TrackingTimeline";

interface PublicTrackingResponse {
  tracking_number: string;
  status: string;
  current_location_city?: string;
  current_location_hub?: string;
  estimated_delivery_at?: string;
  delivered_at?: string;
  delivery_attempts: number;
  recipient_name: string;
  recipient_city?: string;
  events: ShipmentEvent[];
}

const API_BASE = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

function formatETA(ts?: string): string {
  if (!ts) return "—";
  try {
    return new Date(ts).toLocaleString("en-GB", {
      day: "2-digit",
      month: "short",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
    });
  } catch {
    return ts;
  }
}

function statusColor(status: string): string {
  if (status === "delivered") return "bg-emerald-100 text-emerald-700 ring-emerald-200";
  if (status === "out_for_delivery") return "bg-amber-100 text-amber-700 ring-amber-200";
  if (status === "returned_to_sender" || status === "returned" || status === "cancelled")
    return "bg-red-100 text-red-700 ring-red-200";
  if (status === "held_at_office" || status === "redirected")
    return "bg-violet-100 text-violet-700 ring-violet-200";
  return "bg-blue-100 text-blue-700 ring-blue-200";
}

export default function PublicTrackingPage() {
  const { trackingNumber } = useParams<{ trackingNumber: string }>();
  const [last4, setLast4] = useState<string>("");
  const [data, setData] = useState<PublicTrackingResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    const stored = sessionStorage.getItem(`tracking_last4_${trackingNumber}`);
    if (stored) {
      setLast4(stored);
      lookup(stored);
    }
  }, [trackingNumber]);

  async function lookup(code?: string) {
    const c = code ?? last4;
    if (!c || c.length !== 4) {
      setError("Enter last 4 digits of recipient phone");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const r = await fetch(
        `${API_BASE}/api/v1/public/tracking/${trackingNumber}?last4=${c}`,
      );
      const j = await r.json();
      if (!r.ok) {
        setError(j?.error?.message || "Verification failed");
        setData(null);
      } else {
        setData(j.data as PublicTrackingResponse);
        sessionStorage.setItem(`tracking_last4_${trackingNumber}`, c);
      }
    } catch {
      setError("Network error");
    } finally {
      setLoading(false);
    }
  }

  return (
    <main className="min-h-screen bg-gradient-to-b from-gray-50 to-white py-10">
      <div className="mx-auto max-w-3xl px-4">
        <header className="mb-8 flex items-center gap-3">
          <div className="flex h-10 w-10 items-center justify-center rounded-xl bg-gradient-to-br from-blue-600 to-violet-600 shadow-md">
            <Truck className="h-5 w-5 text-white" />
          </div>
          <div>
            <h1 className="text-xl font-semibold text-gray-900">ChainOrchestra Tracking</h1>
            <p className="text-xs text-gray-500">Public shipment status</p>
          </div>
        </header>

        <section className="rounded-2xl border border-gray-200 bg-white p-6 shadow-sm">
          <div className="flex items-center gap-2 text-xs text-gray-500">
            <Package className="h-3.5 w-3.5" />
            Tracking number
          </div>
          <div className="mt-1 font-mono text-2xl font-semibold text-gray-900">
            {trackingNumber}
          </div>

          {!data && (
            <div className="mt-6 rounded-lg bg-amber-50 p-4 ring-1 ring-amber-100">
              <div className="flex items-center gap-2 text-xs font-semibold text-amber-800">
                <ShieldCheck className="h-4 w-4" />
                Verify with recipient phone (last 4 digits)
              </div>
              <div className="mt-3 flex gap-2">
                <input
                  type="text"
                  inputMode="numeric"
                  maxLength={4}
                  pattern="[0-9]{4}"
                  placeholder="0000"
                  value={last4}
                  onChange={(e) => setLast4(e.target.value.replace(/\D/g, ""))}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") lookup();
                  }}
                  className="w-32 rounded-lg border border-amber-200 bg-white px-3 py-2 text-center font-mono text-lg tracking-widest focus:border-amber-500 focus:outline-none focus:ring-2 focus:ring-amber-200"
                />
                <button
                  onClick={() => lookup()}
                  disabled={loading}
                  className="inline-flex items-center gap-2 rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white shadow-sm hover:bg-amber-700 disabled:opacity-60"
                >
                  {loading ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Search className="h-4 w-4" />
                  )}
                  Track
                </button>
              </div>
              {error && <p className="mt-3 text-xs text-red-600">{error}</p>}
            </div>
          )}

          {data && (
            <>
              <div className="mt-6 grid grid-cols-2 gap-4 sm:grid-cols-3">
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">Status</div>
                  <div className="mt-1">
                    <span
                      className={`inline-flex rounded-full px-3 py-1 text-xs font-medium ring-1 ${statusColor(data.status)}`}
                    >
                      {data.status.replace(/_/g, " ")}
                    </span>
                  </div>
                </div>
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">
                    Estimated delivery
                  </div>
                  <div className="mt-1 text-sm text-gray-900">
                    {formatETA(data.estimated_delivery_at)}
                  </div>
                </div>
                <div>
                  <div className="text-xs uppercase tracking-wide text-gray-500">Recipient</div>
                  <div className="mt-1 text-sm text-gray-900">
                    {data.recipient_name}
                    {data.recipient_city && (
                      <span className="text-gray-500"> · {data.recipient_city}</span>
                    )}
                  </div>
                </div>
                {(data.current_location_city || data.current_location_hub) && (
                  <div className="col-span-2 sm:col-span-3">
                    <div className="text-xs uppercase tracking-wide text-gray-500">
                      Current location
                    </div>
                    <div className="mt-1 text-sm text-gray-900">
                      {data.current_location_hub
                        ? `${data.current_location_hub}${data.current_location_city ? ` · ${data.current_location_city}` : ""}`
                        : data.current_location_city}
                    </div>
                  </div>
                )}
                {data.delivered_at && (
                  <div>
                    <div className="text-xs uppercase tracking-wide text-gray-500">
                      Delivered at
                    </div>
                    <div className="mt-1 text-sm text-gray-900">
                      {formatETA(data.delivered_at)}
                    </div>
                  </div>
                )}
                {data.delivery_attempts > 0 && (
                  <div>
                    <div className="text-xs uppercase tracking-wide text-gray-500">
                      Delivery attempts
                    </div>
                    <div className="mt-1 text-sm text-gray-900">{data.delivery_attempts}</div>
                  </div>
                )}
              </div>

              <h2 className="mt-8 text-sm font-semibold uppercase tracking-wide text-gray-500">
                Timeline
              </h2>
              <div className="mt-4">
                <TrackingTimeline events={data.events} />
              </div>
            </>
          )}
        </section>

        <p className="mt-6 text-center text-xs text-gray-400">
          ChainOrchestra · public tracking · verification by last 4 digits of recipient phone
        </p>
      </div>
    </main>
  );
}
