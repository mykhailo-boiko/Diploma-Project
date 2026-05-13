import { NextResponse } from "next/server";
import type { NextRequest } from "next/server";
import { jwtVerify } from "jose";

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

const SECRET = new TextEncoder().encode(process.env.JWT_SECRET ?? "dev-secret-change-me");

async function verifiedJwtRole(token: string): Promise<string | null> {
  try {
    const { payload } = await jwtVerify(token, SECRET, { algorithms: ["HS256"] });
    return typeof payload.role === "string" ? payload.role : null;
  } catch {
    return null;
  }
}

export async function proxy(request: NextRequest) {
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

  if (isProtected && token) {
    const role = await verifiedJwtRole(token);
    if (role === null) {
      const url = new URL("/login", request.url);
      url.searchParams.set("from", pathname);
      url.searchParams.set("reason", "invalid_token");
      return NextResponse.redirect(url);
    }
    const isAdminPath = ADMIN_PREFIXES.some((p) => pathname.startsWith(p));
    if (isAdminPath && role !== "admin") {
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
