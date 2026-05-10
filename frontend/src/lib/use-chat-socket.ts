"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useAuthStore } from "@/stores/auth";

export type MessageType =
  | "message"
  | "thinking"
  | "tool_start"
  | "tool_result"
  | "tool_error"
  | "partial_failure"
  | "stream"
  | "system"
  | "plan"
  | "error";

export interface ChatMessage {
  id: string;
  role: "user" | "assistant" | "system";
  type: MessageType;
  content: string;
  timestamp: number;
  planId?: string;
}

export interface PlanStep {
  id: string;
  tool: string;
  params: Record<string, unknown>;
  result: Record<string, unknown> | null;
  status: "pending" | "running" | "success" | "failed" | "skipped";
  started_at: string | null;
  finished_at: string | null;
  duration_ms: number | null;
  error: string | null;
}

export interface ExecutionPlan {
  id: string;
  session_id: string;
  intent: string;
  steps: PlanStep[];
  status: "running" | "completed" | "partial_failure" | "failed";
  created_at: string;
  finished_at: string | null;
}

type ConnectionStatus = "connecting" | "connected" | "disconnected";

const WS_BASE =
  (process.env.NEXT_PUBLIC_MCP_WS_URL || "ws://localhost:8090") + "/ws/chat";

let _nextId = 0;
function msgId(): string {
  return `msg-${Date.now()}-${++_nextId}`;
}

export function useChatSocket() {
  const token = useAuthStore((s) => s.token);
  const [messages, setMessages] = useState<ChatMessage[]>([]);
  const [plans, setPlans] = useState<Record<string, ExecutionPlan>>({});
  const [status, setStatus] = useState<ConnectionStatus>("disconnected");
  const [isThinking, setIsThinking] = useState(false);
  const wsRef = useRef<WebSocket | null>(null);
  const reconnectTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const streamBuf = useRef<string>("");
  const streamMsgId = useRef<string | null>(null);
  const connectRef = useRef<() => void>(() => {});

  const connect = useCallback(() => {
    if (!token) return;
    if (wsRef.current?.readyState === WebSocket.OPEN) return;

    setStatus("connecting");
    const ws = new WebSocket(`${WS_BASE}?token=${token}`);
    wsRef.current = ws;

    ws.onopen = () => setStatus("connected");

    ws.onmessage = (ev) => {
      try {
        const data = JSON.parse(ev.data) as {
          type: MessageType;
          content: string;
        };
        const { type, content } = data;

        if (type === "thinking") {
          setIsThinking(true);
          return;
        }

        if (type === "stream") {
          setIsThinking(false);
          streamBuf.current += content;
          const id = streamMsgId.current ?? msgId();
          streamMsgId.current = id;
          setMessages((prev) => {
            const idx = prev.findIndex((m) => m.id === id);
            const updated: ChatMessage = {
              id,
              role: "assistant",
              type: "stream",
              content: streamBuf.current,
              timestamp: Date.now(),
            };
            if (idx >= 0) {
              const copy = [...prev];
              copy[idx] = updated;
              return copy;
            }
            return [...prev, updated];
          });
          return;
        }

        if (type === "message") {
          setIsThinking(false);
          if (streamMsgId.current) {
            const sid = streamMsgId.current;
            streamBuf.current = "";
            streamMsgId.current = null;
            setMessages((prev) => {
              const idx = prev.findIndex((m) => m.id === sid);
              const finalMsg: ChatMessage = {
                id: sid,
                role: "assistant",
                type: "message",
                content,
                timestamp: Date.now(),
              };
              if (idx >= 0) {
                const copy = [...prev];
                copy[idx] = finalMsg;
                return copy;
              }
              return [...prev, finalMsg];
            });
          } else {
            setMessages((prev) => [
              ...prev,
              {
                id: msgId(),
                role: "assistant",
                type: "message",
                content,
                timestamp: Date.now(),
              },
            ]);
          }
          return;
        }

        if (type === "system") {
          setIsThinking(false);
          setMessages((prev) => [
            ...prev,
            { id: msgId(), role: "system", type, content, timestamp: Date.now() },
          ]);
          return;
        }

        if (type === "plan") {
          try {
            const plan = JSON.parse(content) as ExecutionPlan;
            setPlans((prev) => ({ ...prev, [plan.id]: plan }));
            setMessages((prev) => {
              const lastAssistantIdx = [...prev].reverse().findIndex(
                (m) => m.role === "assistant" && (m.type === "message" || m.type === "stream"),
              );
              if (lastAssistantIdx < 0) return prev;
              const idx = prev.length - 1 - lastAssistantIdx;
              const copy = [...prev];
              copy[idx] = { ...copy[idx], planId: plan.id };
              return copy;
            });
          } catch {
            // ignore malformed plan payload
          }
          return;
        }

        setIsThinking(false);
        setMessages((prev) => [
          ...prev,
          {
            id: msgId(),
            role: "assistant",
            type,
            content,
            timestamp: Date.now(),
          },
        ]);
      } catch {
      }
    };

    ws.onclose = () => {
      setStatus("disconnected");
      setIsThinking(false);
      wsRef.current = null;
      reconnectTimer.current = setTimeout(() => connectRef.current(), 3000);
    };

    ws.onerror = () => {
      ws.close();
    };
  }, [token]);

  useEffect(() => {
    connectRef.current = connect;
  }, [connect]);

  const disconnect = useCallback(() => {
    if (reconnectTimer.current) {
      clearTimeout(reconnectTimer.current);
      reconnectTimer.current = null;
    }
    wsRef.current?.close();
    wsRef.current = null;
    setStatus("disconnected");
  }, []);

  const sendMessage = useCallback(
    (text: string) => {
      if (!wsRef.current || wsRef.current.readyState !== WebSocket.OPEN) return;

      streamBuf.current = "";
      streamMsgId.current = null;

      setMessages((prev) => [
        ...prev,
        {
          id: msgId(),
          role: "user",
          type: "message",
          content: text,
          timestamp: Date.now(),
        },
      ]);

      wsRef.current.send(JSON.stringify({ message: text }));
    },
    [],
  );

  const clearMessages = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify({ message: "/clear" }));
    }
    setMessages([]);
    setPlans({});
  }, []);

  useEffect(() => {
    return () => disconnect();
  }, [disconnect]);

  return {
    messages,
    plans,
    status,
    isThinking,
    connect,
    disconnect,
    sendMessage,
    clearMessages,
  };
}
