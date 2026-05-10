export function safeFixed(
  n: number | null | undefined,
  digits = 2,
  fallback = "—",
): string {
  return typeof n === "number" && Number.isFinite(n) ? n.toFixed(digits) : fallback;
}

export function safeLocale(
  n: number | null | undefined,
  fallback = "0",
): string {
  return typeof n === "number" && Number.isFinite(n) ? n.toLocaleString() : fallback;
}

export function safeCurrency(
  n: number | null | undefined,
  symbol = "$",
  digits = 2,
): string {
  return typeof n === "number" && Number.isFinite(n)
    ? `${symbol}${n.toFixed(digits)}`
    : "—";
}

export function safePercent(
  n: number | null | undefined,
  digits = 1,
): string {
  return typeof n === "number" && Number.isFinite(n) ? `${n.toFixed(digits)}%` : "—";
}
