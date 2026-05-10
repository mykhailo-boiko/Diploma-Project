"use client";

import { useState, useRef, useEffect } from "react";
import { useRouter } from "next/navigation";
import { Bell, LogOut, User, Sparkles } from "lucide-react";
import { useAuthStore } from "@/stores/auth";
import { formatRole } from "@/lib/roles";
import { useUnreadCount } from "@/lib/use-notifications";
import { clsx } from "clsx";

interface TopbarProps {
  onToggleChat?: () => void;
  chatOpen?: boolean;
}

export default function Topbar({ onToggleChat, chatOpen }: TopbarProps) {
  const router = useRouter();
  const { user, logout } = useAuthStore();
  const [menuOpen, setMenuOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) {
        setMenuOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  function handleLogout() {
    logout();
    router.replace("/login");
  }

  const unreadCount = useUnreadCount();
  const unread = unreadCount.data?.data?.unread_count ?? 0;

  if (!user) return null;

  return (
    <header className="flex h-16 items-center justify-between border-b border-gray-200 bg-white px-6">
      <div />

      <div className="flex items-center gap-4">
        <button
          onClick={onToggleChat}
          className={clsx(
            "group relative inline-flex items-center gap-2 overflow-hidden rounded-full px-3 py-1.5 text-sm font-semibold text-white shadow-md shadow-blue-500/20 transition-all duration-200 hover:shadow-lg hover:shadow-blue-500/30 hover:-translate-y-0.5 focus:outline-none focus:ring-2 focus:ring-blue-400 focus:ring-offset-2",
            chatOpen
              ? "bg-gradient-to-r from-blue-600 via-indigo-600 to-purple-600"
              : "bg-gradient-to-r from-blue-500 via-indigo-500 to-purple-500",
          )}
          aria-label="AI Assistant"
          title="AI Assistant — chat with your supply chain"
        >
          {!chatOpen && (
            <span className="pointer-events-none absolute inset-0 -z-10 animate-pulse rounded-full bg-gradient-to-r from-blue-400 via-indigo-400 to-purple-400 opacity-60 blur-md" />
          )}
          <Sparkles className="h-4 w-4 drop-shadow-sm" />
          <span className="hidden sm:inline">AI Assistant</span>
        </button>

        <button
          onClick={() => router.push("/notifications")}
          className="relative rounded-md p-2 text-gray-500 hover:bg-gray-100 hover:text-gray-700"
          aria-label="Notifications"
        >
          <Bell className="h-5 w-5" />
          {unread > 0 && (
            <span className="absolute -top-0.5 -right-0.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-red-500 px-1 text-[10px] font-bold text-white">
              {unread > 99 ? "99+" : unread}
            </span>
          )}
        </button>

        <div className="relative" ref={menuRef}>
          <button
            onClick={() => setMenuOpen(!menuOpen)}
            className="flex items-center gap-2 rounded-md px-3 py-1.5 text-sm hover:bg-gray-100"
          >
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-blue-100 text-blue-700">
              <User className="h-4 w-4" />
            </div>
            <div className="hidden text-left md:block">
              <div className="text-sm font-medium text-gray-900">
                {user.first_name} {user.last_name}
              </div>
              <div className="text-xs text-gray-500">
                {formatRole(user.role)}
              </div>
            </div>
          </button>

          {menuOpen && (
            <div className="absolute right-0 mt-1 w-48 rounded-md border border-gray-200 bg-white py-1 shadow-lg">
              <div className="border-b border-gray-100 px-4 py-2 md:hidden">
                <div className="text-sm font-medium text-gray-900">
                  {user.first_name} {user.last_name}
                </div>
                <div className="text-xs text-gray-500">
                  {formatRole(user.role)}
                </div>
              </div>
              <button
                onClick={handleLogout}
                className={clsx(
                  "flex w-full items-center gap-2 px-4 py-2 text-sm text-gray-700 hover:bg-gray-100",
                )}
              >
                <LogOut className="h-4 w-4" />
                Sign out
              </button>
            </div>
          )}
        </div>
      </div>
    </header>
  );
}
