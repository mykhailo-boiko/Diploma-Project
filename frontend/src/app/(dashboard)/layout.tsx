"use client";

import { useState } from "react";
import AuthGuard from "@/components/AuthGuard";
import Sidebar from "@/components/Sidebar";
import Topbar from "@/components/Topbar";
import ChatPanel from "@/components/ChatPanel";
import ActivityFeed from "@/components/ActivityFeed";

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [collapsed, setCollapsed] = useState(false);
  const [chatOpen, setChatOpen] = useState(false);
  const [feedOpen, setFeedOpen] = useState(false);

  return (
    <AuthGuard>
      <div className="flex h-screen overflow-hidden">
        <Sidebar
          collapsed={collapsed}
          onToggle={() => setCollapsed(!collapsed)}
        />
        <div className="flex flex-1 flex-col overflow-hidden">
          <Topbar
            onToggleChat={() => setChatOpen((v) => !v)}
            chatOpen={chatOpen}
            onToggleFeed={() => setFeedOpen((v) => !v)}
          />
          <main className="flex-1 overflow-y-auto bg-gray-50 p-6">
            {children}
          </main>
        </div>
        {feedOpen && <ActivityFeed onClose={() => setFeedOpen(false)} />}
        <ChatPanel open={chatOpen} onClose={() => setChatOpen(false)} />
      </div>
    </AuthGuard>
  );
}
