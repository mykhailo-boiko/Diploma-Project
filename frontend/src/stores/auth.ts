"use client";

import { create } from "zustand";
import { apiFetch } from "@/lib/api";
import { LoginResponse, User } from "@/lib/types";

interface AuthState {
  user: User | null;
  token: string | null;
  isLoading: boolean;
  isAuthenticated: boolean;

  login: (email: string, password: string) => Promise<void>;
  logout: () => void;
  hydrate: () => void;
}

function decodeJwtUser(token: string): User | null {
  try {
    const payloadB64 = token.split(".")[1];
    if (!payloadB64) return null;
    const padded = payloadB64 + "=".repeat((4 - (payloadB64.length % 4)) % 4);
    const json = atob(padded.replace(/-/g, "+").replace(/_/g, "/"));
    const claims = JSON.parse(json) as {
      user_id?: string;
      sub?: string;
      email?: string;
      role?: string;
      first_name?: string;
      last_name?: string;
    };
    const id = claims.user_id || claims.sub;
    if (!id || !claims.email || !claims.role) return null;
    return {
      id,
      email: claims.email,
      role: claims.role,
      first_name: claims.first_name || "",
      last_name: claims.last_name || "",
    } as User;
  } catch {
    return null;
  }
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  token: null,
  isLoading: true,
  isAuthenticated: false,

  login: async (email: string, password: string) => {
    const res = await apiFetch<LoginResponse>("/api/v1/auth/login", {
      method: "POST",
      body: { email, password },
    });

    const { access_token, refresh_token } = res.data;
    let user: User | null = (res.data as unknown as { user?: User }).user ?? null;

    if (!user) {
      user = decodeJwtUser(access_token);
    }

    if (!user) {
      try {
        const me = await apiFetch<{ data: User }>("/api/v1/users/me", {
          method: "GET",
          headers: { Authorization: `Bearer ${access_token}` },
        });
        user = me.data ?? null;
      } catch {
        user = null;
      }
    }

    localStorage.setItem("access_token", access_token);
    localStorage.setItem("refresh_token", refresh_token);
    if (user) localStorage.setItem("user", JSON.stringify(user));
    document.cookie = `access_token=${encodeURIComponent(access_token)}; path=/; max-age=900; samesite=lax`;
    if (user?.role) {
      document.cookie = `user_role=${encodeURIComponent(user.role)}; path=/; max-age=900; samesite=lax`;
    }

    set({ user, token: access_token, isAuthenticated: true, isLoading: false });
  },

  logout: () => {
    localStorage.removeItem("access_token");
    localStorage.removeItem("refresh_token");
    localStorage.removeItem("user");
    document.cookie = "access_token=; path=/; max-age=0";
    document.cookie = "user_role=; path=/; max-age=0";
    set({ user: null, token: null, isAuthenticated: false, isLoading: false });
  },

  hydrate: () => {
    const token = localStorage.getItem("access_token");
    const userJson = localStorage.getItem("user");

    if (!token) {
      set({ isLoading: false });
      return;
    }

    let user: User | null = null;
    if (userJson && userJson !== "undefined" && userJson !== "null") {
      try {
        user = JSON.parse(userJson) as User;
      } catch {
        user = null;
      }
    }

    if (!user) {
      user = decodeJwtUser(token);
      if (user) localStorage.setItem("user", JSON.stringify(user));
    }

    if (user) {
      set({ user, token, isAuthenticated: true, isLoading: false });
    } else {
      localStorage.removeItem("access_token");
      localStorage.removeItem("refresh_token");
      localStorage.removeItem("user");
      set({ isLoading: false });
    }
  },
}));
