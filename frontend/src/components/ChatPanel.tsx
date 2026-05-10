"use client";

import { useEffect, useRef, useState, useCallback } from "react";
import {
  X,
  Send,
  Loader2,
  Trash2,
  WifiOff,
  Wrench,
  AlertCircle,
  CheckCircle2,
  Info,
  Maximize2,
  Minimize2,
  PanelRightClose,
  Sparkles,
  ChevronsLeft,
  ChevronsRight,
} from "lucide-react";
import clsx from "clsx";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useChatSocket, ChatMessage, MessageType } from "@/lib/use-chat-socket";

interface ChatPanelProps {
  open: boolean;
  onClose: () => void;
}

type WidthMode = "compact" | "wide" | "fullscreen";

const STORAGE_KEY = "chatpanel.widthMode";
const CUSTOM_WIDTH_KEY = "chatpanel.customWidth";

function getModeFromStorage(): WidthMode {
  if (typeof window === "undefined") return "wide";
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "compact" || saved === "wide" || saved === "fullscreen") return saved;
  return "wide";
}

function getCustomWidth(): number | null {
  if (typeof window === "undefined") return null;
  const v = localStorage.getItem(CUSTOM_WIDTH_KEY);
  return v ? parseInt(v, 10) : null;
}

function StatusDot({ status }: { status: string }) {
  const color =
    status === "connected"
      ? "bg-green-500"
      : status === "connecting"
        ? "bg-yellow-500 animate-pulse"
        : "bg-gray-400";
  return <span className={clsx("inline-block h-2 w-2 rounded-full", color)} />;
}

function ToolBadge({ type, content }: { type: MessageType; content: string }) {
  if (type === "tool_start") {
    return (
      <div className="inline-flex items-center gap-2 rounded-full bg-blue-50 px-3 py-1 text-xs font-medium text-blue-700 ring-1 ring-blue-100">
        <Wrench className="h-3 w-3" />
        <span>{content}</span>
      </div>
    );
  }
  if (type === "tool_result") {
    return (
      <div className="inline-flex items-center gap-2 rounded-full bg-emerald-50 px-3 py-1 text-xs font-medium text-emerald-700 ring-1 ring-emerald-100">
        <CheckCircle2 className="h-3 w-3" />
        <span>{content}</span>
      </div>
    );
  }
  if (type === "tool_error" || type === "partial_failure") {
    return (
      <div className="inline-flex items-center gap-2 rounded-full bg-red-50 px-3 py-1 text-xs font-medium text-red-700 ring-1 ring-red-100">
        <AlertCircle className="h-3 w-3" />
        <span>{content}</span>
      </div>
    );
  }
  return null;
}

function MessageBubble({ msg }: { msg: ChatMessage }) {
  if (msg.role === "system") {
    return (
      <div className="flex justify-center">
        <div className="flex items-center gap-1.5 rounded-full bg-gray-100 px-3 py-1 text-xs text-gray-500">
          <Info className="h-3 w-3" />
          {msg.content}
        </div>
      </div>
    );
  }

  if (
    msg.type === "tool_start" ||
    msg.type === "tool_result" ||
    msg.type === "tool_error" ||
    msg.type === "partial_failure"
  ) {
    return (
      <div className="flex justify-start pl-1">
        <ToolBadge type={msg.type} content={msg.content} />
      </div>
    );
  }

  if (msg.type === "error") {
    return (
      <div className="flex justify-start pl-1">
        <div className="flex items-start gap-2 rounded-xl bg-red-50 px-3 py-2 text-sm text-red-700 ring-1 ring-red-100">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{msg.content}</span>
        </div>
      </div>
    );
  }

  const isUser = msg.role === "user";

  if (isUser) {
    return (
      <div className="flex justify-end">
        <div className="max-w-[80%] rounded-2xl rounded-br-md bg-blue-600 px-4 py-2.5 text-sm text-white shadow-sm">
          <p className="whitespace-pre-wrap break-words">{msg.content}</p>
        </div>
      </div>
    );
  }

  return (
    <div className="flex w-full justify-start gap-2.5">
      <div className="mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-blue-600 shadow-sm">
        <Sparkles className="h-3.5 w-3.5 text-white" />
      </div>
      <div className="min-w-0 flex-1">
        <div className="rounded-2xl rounded-tl-md bg-white px-4 py-3 text-sm text-gray-800 shadow-sm ring-1 ring-gray-100">
          <div
            className={clsx(
              "prose prose-sm max-w-none break-words",
              "prose-headings:my-2 prose-headings:font-semibold",
              "prose-p:my-1.5 prose-p:leading-relaxed",
              "prose-ul:my-1.5 prose-ol:my-1.5 prose-li:my-0.5",
              "prose-strong:text-gray-900",
              "prose-a:text-blue-600 prose-a:no-underline hover:prose-a:underline",
              "prose-code:rounded prose-code:bg-gray-100 prose-code:px-1 prose-code:py-0.5",
              "prose-code:text-xs prose-code:font-mono prose-code:text-gray-800",
              "prose-code:before:content-none prose-code:after:content-none",
              "prose-pre:my-2 prose-pre:rounded-lg prose-pre:bg-gray-900",
              "prose-pre:px-3 prose-pre:py-2 prose-pre:text-xs",
            )}
          >
            <ReactMarkdown
              remarkPlugins={[remarkGfm]}
              components={{
                table: ({ children }) => (
                  <div className="my-2 overflow-x-auto rounded-lg ring-1 ring-gray-200">
                    <table className="min-w-full border-collapse text-xs">
                      {children}
                    </table>
                  </div>
                ),
                thead: ({ children }) => (
                  <thead className="bg-gray-50">{children}</thead>
                ),
                th: ({ children }) => (
                  <th className="whitespace-nowrap border-b border-gray-200 px-3 py-2 text-left text-[11px] font-semibold uppercase tracking-wide text-gray-600">
                    {children}
                  </th>
                ),
                td: ({ children }) => {
                  const text = String(children ?? "");
                  // Truncate long UUIDs / IDs visually but keep full on hover
                  const isUuid = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(text.trim());
                  if (isUuid) {
                    return (
                      <td
                        className="whitespace-nowrap border-b border-gray-100 px-3 py-2 font-mono text-[11px] text-gray-500"
                        title={text}
                      >
                        {text.slice(0, 8)}…
                      </td>
                    );
                  }
                  return (
                    <td className="whitespace-nowrap border-b border-gray-100 px-3 py-2 text-gray-700">
                      {children}
                    </td>
                  );
                },
                tr: ({ children }) => (
                  <tr className="hover:bg-gray-50">{children}</tr>
                ),
              }}
            >
              {msg.content}
            </ReactMarkdown>
          </div>
        </div>
      </div>
    </div>
  );
}

export default function ChatPanel({ open, onClose }: ChatPanelProps) {
  const {
    messages,
    status,
    isThinking,
    connect,
    disconnect,
    sendMessage,
    clearMessages,
  } = useChatSocket();

  const [input, setInput] = useState("");
  const [widthMode, setWidthMode] = useState<WidthMode>("wide");
  const [customWidth, setCustomWidth] = useState<number | null>(null);
  const [isResizing, setIsResizing] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);
  const panelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    setWidthMode(getModeFromStorage());
    setCustomWidth(getCustomWidth());
  }, []);

  useEffect(() => {
    if (open) {
      connect();
      setTimeout(() => inputRef.current?.focus(), 200);
    } else {
      disconnect();
    }
  }, [open, connect, disconnect]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [messages, isThinking]);

  // Drag resize
  const startResize = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    setIsResizing(true);
  }, []);

  useEffect(() => {
    if (!isResizing) return;
    function onMouseMove(e: MouseEvent) {
      const w = Math.max(360, Math.min(window.innerWidth - 80, window.innerWidth - e.clientX));
      setCustomWidth(w);
      setWidthMode("compact"); // mark as custom-controlled
    }
    function onMouseUp() {
      setIsResizing(false);
      if (customWidth) {
        localStorage.setItem(CUSTOM_WIDTH_KEY, String(customWidth));
      }
    }
    document.addEventListener("mousemove", onMouseMove);
    document.addEventListener("mouseup", onMouseUp);
    return () => {
      document.removeEventListener("mousemove", onMouseMove);
      document.removeEventListener("mouseup", onMouseUp);
    };
  }, [isResizing, customWidth]);

  function setMode(m: WidthMode) {
    setWidthMode(m);
    setCustomWidth(null);
    localStorage.setItem(STORAGE_KEY, m);
    localStorage.removeItem(CUSTOM_WIDTH_KEY);
  }

  function handleSend() {
    const text = input.trim();
    if (!text || status !== "connected") return;
    sendMessage(text);
    setInput("");
    if (inputRef.current) {
      inputRef.current.style.height = "auto";
    }
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (e.key === "Enter" && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  }

  function handleInput(e: React.ChangeEvent<HTMLTextAreaElement>) {
    setInput(e.target.value);
    const el = e.target;
    el.style.height = "auto";
    el.style.height = Math.min(el.scrollHeight, 160) + "px";
  }

  // Compute panel width
  let panelStyle: React.CSSProperties = {};
  if (widthMode === "fullscreen") {
    panelStyle.width = "min(1280px, 95vw)";
  } else if (customWidth) {
    panelStyle.width = `${customWidth}px`;
  } else if (widthMode === "wide") {
    panelStyle.width = "min(720px, 60vw)";
  } else {
    panelStyle.width = "min(440px, 90vw)";
  }

  const SUGGESTIONS = [
    "Show all orders",
    "Which products have low stock?",
    "Sales trends for the last 30 days",
    "Show pending shipments",
    "Top 5 customers by revenue",
  ];

  return (
    <>
      {open && (
        <div
          className={clsx(
            "fixed inset-0 z-40 transition-opacity",
            widthMode === "fullscreen" ? "bg-black/40" : "bg-black/10 md:bg-transparent",
          )}
          onClick={onClose}
        />
      )}

      <div
        ref={panelRef}
        style={panelStyle}
        className={clsx(
          "fixed right-0 top-0 z-50 flex h-full flex-col border-l border-gray-200 bg-gradient-to-b from-gray-50 to-white shadow-2xl transition-transform duration-300 ease-out",
          open ? "translate-x-0" : "translate-x-full",
          isResizing && "select-none",
        )}
      >
        {/* Resize handle */}
        <div
          onMouseDown={startResize}
          className="group absolute left-0 top-0 z-10 flex h-full w-1.5 cursor-col-resize items-center justify-center hover:bg-blue-500/20"
          title="Drag to resize"
        >
          <div className="h-12 w-1 rounded-full bg-gray-300 opacity-0 transition-opacity group-hover:opacity-100" />
        </div>

        {/* Header */}
        <div className="flex h-14 items-center justify-between border-b border-gray-200 bg-white/80 px-4 backdrop-blur">
          <div className="flex items-center gap-2.5">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-gradient-to-br from-violet-500 to-blue-600 shadow">
              <Sparkles className="h-4 w-4 text-white" />
            </div>
            <div className="flex flex-col">
              <h2 className="text-sm font-semibold text-gray-900">AI Assistant</h2>
              <div className="flex items-center gap-1.5 text-[11px] text-gray-500">
                <StatusDot status={status} />
                <span className="capitalize">{status}</span>
              </div>
            </div>
          </div>
          <div className="flex items-center gap-1">
            {/* Width controls */}
            <div className="mr-1 hidden items-center gap-0.5 rounded-md bg-gray-100 p-0.5 md:flex">
              <button
                onClick={() => setMode("compact")}
                className={clsx(
                  "rounded p-1 text-gray-500 hover:bg-white hover:text-gray-700",
                  widthMode === "compact" && !customWidth && "bg-white text-gray-900 shadow-sm",
                )}
                title="Compact"
              >
                <ChevronsRight className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => setMode("wide")}
                className={clsx(
                  "rounded p-1 text-gray-500 hover:bg-white hover:text-gray-700",
                  widthMode === "wide" && !customWidth && "bg-white text-gray-900 shadow-sm",
                )}
                title="Wide"
              >
                <ChevronsLeft className="h-3.5 w-3.5" />
              </button>
              <button
                onClick={() => setMode(widthMode === "fullscreen" ? "wide" : "fullscreen")}
                className={clsx(
                  "rounded p-1 text-gray-500 hover:bg-white hover:text-gray-700",
                  widthMode === "fullscreen" && "bg-white text-gray-900 shadow-sm",
                )}
                title={widthMode === "fullscreen" ? "Exit fullscreen" : "Fullscreen"}
              >
                {widthMode === "fullscreen" ? (
                  <Minimize2 className="h-3.5 w-3.5" />
                ) : (
                  <Maximize2 className="h-3.5 w-3.5" />
                )}
              </button>
            </div>

            {status === "disconnected" && (
              <button
                onClick={connect}
                className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                title="Reconnect"
              >
                <WifiOff className="h-4 w-4" />
              </button>
            )}
            {status === "connected" && messages.length > 0 && (
              <button
                onClick={clearMessages}
                className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                title="Clear conversation"
              >
                <Trash2 className="h-4 w-4" />
              </button>
            )}
            <button
              onClick={onClose}
              className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
              title="Close"
            >
              <PanelRightClose className="h-4 w-4 hidden md:block" />
              <X className="h-4 w-4 md:hidden" />
            </button>
          </div>
        </div>

        {/* Messages */}
        <div className="flex-1 overflow-y-auto px-4 py-4">
          <div className="mx-auto flex max-w-4xl flex-col gap-4">
            {messages.length === 0 && status === "connected" && (
              <div className="mt-12 text-center">
                <div className="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-blue-600 shadow-md">
                  <Sparkles className="h-6 w-6 text-white" />
                </div>
                <h3 className="mt-3 text-base font-semibold text-gray-900">
                  How can I help today?
                </h3>
                <p className="mt-1 text-xs text-gray-500">
                  Ask anything about orders, inventory, logistics, or analytics.
                </p>
                <div className="mx-auto mt-6 flex max-w-md flex-wrap justify-center gap-2">
                  {SUGGESTIONS.map((s) => (
                    <button
                      key={s}
                      onClick={() => {
                        setInput(s);
                        setTimeout(() => inputRef.current?.focus(), 0);
                      }}
                      className="rounded-full border border-gray-200 bg-white px-3 py-1.5 text-xs text-gray-700 shadow-sm transition hover:border-blue-300 hover:bg-blue-50 hover:text-blue-700"
                    >
                      {s}
                    </button>
                  ))}
                </div>
              </div>
            )}

            {messages.map((msg) => (
              <MessageBubble key={msg.id} msg={msg} />
            ))}

            {isThinking && (
              <div className="flex w-full justify-start gap-2.5">
                <div className="mt-1 flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-gradient-to-br from-violet-500 to-blue-600 shadow-sm">
                  <Sparkles className="h-3.5 w-3.5 text-white" />
                </div>
                <div className="flex items-center gap-2 rounded-2xl rounded-tl-md bg-white px-4 py-2.5 text-sm text-gray-500 shadow-sm ring-1 ring-gray-100">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  <span>Thinking…</span>
                </div>
              </div>
            )}

            <div ref={bottomRef} />
          </div>
        </div>

        {/* Input */}
        <div className="border-t border-gray-200 bg-white p-3">
          <div className="mx-auto max-w-4xl">
            {status !== "connected" ? (
              <div className="flex items-center justify-center gap-2 rounded-md bg-gray-50 py-2 text-xs text-gray-500">
                {status === "connecting" ? (
                  <>
                    <Loader2 className="h-3 w-3 animate-spin" />
                    Connecting…
                  </>
                ) : (
                  <>
                    <WifiOff className="h-3 w-3" />
                    Disconnected
                    <button
                      onClick={connect}
                      className="ml-1 font-medium text-blue-600 hover:underline"
                    >
                      Reconnect
                    </button>
                  </>
                )}
              </div>
            ) : (
              <div className="flex items-end gap-2 rounded-2xl border border-gray-200 bg-white p-2 shadow-sm focus-within:border-blue-500 focus-within:ring-1 focus-within:ring-blue-500">
                <textarea
                  ref={inputRef}
                  value={input}
                  onChange={handleInput}
                  onKeyDown={handleKeyDown}
                  placeholder="Type a message…  (Enter to send, Shift+Enter for newline)"
                  rows={1}
                  className="max-h-[160px] min-h-[28px] flex-1 resize-none border-0 bg-transparent px-2 py-1 text-sm focus:outline-none focus:ring-0"
                />
                <button
                  onClick={handleSend}
                  disabled={!input.trim()}
                  className={clsx(
                    "flex h-9 w-9 shrink-0 items-center justify-center rounded-xl transition-all",
                    input.trim()
                      ? "bg-gradient-to-br from-violet-500 to-blue-600 text-white shadow hover:scale-[1.03]"
                      : "bg-gray-100 text-gray-400",
                  )}
                  title="Send (Enter)"
                >
                  <Send className="h-4 w-4" />
                </button>
              </div>
            )}
            <p className="mt-1.5 px-2 text-[10px] text-gray-400">
              AI responses may contain errors. Verify important information.
            </p>
          </div>
        </div>
      </div>
    </>
  );
}
