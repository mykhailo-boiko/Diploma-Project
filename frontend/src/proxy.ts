import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";

const PROTECTED_PREFIXES = [
  "/dashboard",
  "/orders",
  "/shipments",
  "/products",
  "/stock",
  "/admin",
  "/inventory",
  "/analytics",
  "/notifications",
];

const ADMIN_PREFIXES = ["/admin"];

const PUBLIC_PREFIXES = ["/login", "/track", "/_next", "/favicon", "/api/health", "/public"];

function decodeJwtRole(token: string): string | null {
  try {
    const parts = token.split(".");
    if (parts.length < 2) return null;
    const padded = parts[1].replace(/-/g, "+").replace(/_/g, "/");
    const json = JSON.parse(Buffer.from(padded, "base64").toString("utf-8"));
    return typeof json.role === "string" ? json.role : null;
  } catch {
    return null;
  }
}

export function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;

  if (PUBLIC_PREFIXES.some((p) => pathname.startsWith(p))) {
    return NextResponse.next();
  }

  const token = request.cookies.get("access_token")?.value;
  const isProtected = PROTECTED_PREFIXES.some((p) => pathname.startsWith(p));

  if (isProtected && !token) {
    const url = new URL("/login", request.url);
    url.searchParams.set("from", pathname);
    return NextResponse.redirect(url);
  }

  const isAdminPath = ADMIN_PREFIXES.some((p) => pathname.startsWith(p));
  if (isAdminPath && token) {
    const cookieRole = request.cookies.get("user_role")?.value;
    const role = cookieRole || decodeJwtRole(token);
    if (role !== "admin") {
      return NextResponse.redirect(new URL("/dashboard", request.url));
    }
  }

  return NextResponse.next();
}

export const config = {
  matcher: [
    "/dashboard/:path*",
    "/orders/:path*",
    "/shipments/:path*",
    "/products/:path*",
    "/stock/:path*",
    "/admin/:path*",
    "/inventory/:path*",
    "/analytics/:path*",
    "/notifications/:path*",
  ],
};
