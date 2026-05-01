"use client";

import { useEffect, useRef, useState } from "react";
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
} from "lucide-react";
import clsx from "clsx";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useChatSocket, ChatMessage, MessageType } from "@/lib/use-chat-socket";

interface ChatPanelProps {
  open: boolean;
  onClose: () => void;
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

function ToolBadge({
  type,
  content,
}: {
  type: MessageType;
  content: string;
}) {
  if (type === "tool_start") {
    return (
      <div className="flex items-center gap-2 rounded-md bg-blue-50 px-3 py-1.5 text-xs text-blue-700">
        <Wrench className="h-3 w-3" />
        <span>{content}</span>
      </div>
    );
  }
  if (type === "tool_result") {
    return (
      <div className="flex items-center gap-2 rounded-md bg-green-50 px-3 py-1.5 text-xs text-green-700">
        <CheckCircle2 className="h-3 w-3" />
        <span>{content}</span>
      </div>
    );
  }
  if (type === "tool_error" || type === "partial_failure") {
    return (
      <div className="flex items-center gap-2 rounded-md bg-red-50 px-3 py-1.5 text-xs text-red-700">
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
      <div className="flex justify-start pl-2">
        <ToolBadge type={msg.type} content={msg.content} />
      </div>
    );
  }

  if (msg.type === "error") {
    return (
      <div className="flex justify-start pl-2">
        <div className="flex items-start gap-2 rounded-lg bg-red-50 px-3 py-2 text-sm text-red-700">
          <AlertCircle className="mt-0.5 h-4 w-4 shrink-0" />
          <span>{msg.content}</span>
        </div>
      </div>
    );
  }

  const isUser = msg.role === "user";

  return (
    <div className={clsx("flex", isUser ? "justify-end" : "justify-start")}>
      <div
        className={clsx(
          "max-w-[85%] rounded-lg px-3 py-2 text-sm",
          isUser
            ? "bg-blue-600 text-white"
            : "bg-white text-gray-800 shadow-sm border border-gray-100",
        )}
      >
        {isUser ? (
          <p className="whitespace-pre-wrap">{msg.content}</p>
        ) : (
          <div className="prose prose-sm max-w-none prose-p:my-1 prose-headings:my-2 prose-ul:my-1 prose-ol:my-1 prose-pre:my-1 prose-table:my-1 prose-code:bg-gray-100 prose-code:px-1 prose-code:rounded prose-code:text-gray-800 prose-code:before:content-none prose-code:after:content-none">
            <ReactMarkdown remarkPlugins={[remarkGfm]}>
              {msg.content}
            </ReactMarkdown>
          </div>
        )}
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
  const bottomRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLTextAreaElement>(null);

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
    el.style.height = Math.min(el.scrollHeight, 120) + "px";
  }

  return (
    <>
      {}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/20 md:hidden"
          onClick={onClose}
        />
      )}

      {}
      <div
        className={clsx(
          "fixed right-0 top-0 z-50 flex h-full w-full flex-col border-l border-gray-200 bg-gray-50 shadow-xl transition-transform duration-300 ease-in-out md:w-[420px]",
          open ? "translate-x-0" : "translate-x-full",
        )}
      >
        {}
        <div className="flex h-14 items-center justify-between border-b border-gray-200 bg-white px-4">
          <div className="flex items-center gap-2">
            <h2 className="text-sm font-semibold text-gray-900">
              AI Assistant
            </h2>
            <StatusDot status={status} />
          </div>
          <div className="flex items-center gap-1">
            {status === "disconnected" && (
              <button
                onClick={connect}
                className="rounded-md p-1.5 text-gray-400 hover:bg-gray-100 hover:text-gray-600"
                title="Reconnect"
              >
                <WifiOff className="h-4 w-4" />
              </button>
            )}
            {status === "connected" && (
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
            >
              <X className="h-4 w-4" />
            </button>
          </div>
        </div>

        {}
        <div className="flex-1 overflow-y-auto px-4 py-3">
          <div className="flex flex-col gap-3">
            {messages.length === 0 && status === "connected" && (
              <div className="mt-8 text-center text-sm text-gray-400">
                Ask anything about your orders, inventory, logistics, or
                analytics.
              </div>
            )}

            {messages.map((msg) => (
              <MessageBubble key={msg.id} msg={msg} />
            ))}

            {isThinking && (
              <div className="flex justify-start pl-2">
                <div className="flex items-center gap-2 rounded-lg bg-white px-3 py-2 text-sm text-gray-500 shadow-sm border border-gray-100">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" />
                  Thinking...
                </div>
              </div>
            )}

            <div ref={bottomRef} />
          </div>
        </div>

        {}
        <div className="border-t border-gray-200 bg-white p-3">
          {status !== "connected" ? (
            <div className="flex items-center justify-center gap-2 rounded-md bg-gray-50 py-2 text-xs text-gray-500">
              {status === "connecting" ? (
                <>
                  <Loader2 className="h-3 w-3 animate-spin" />
                  Connecting...
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
            <div className="flex items-end gap-2">
              <textarea
                ref={inputRef}
                value={input}
                onChange={handleInput}
                onKeyDown={handleKeyDown}
                placeholder="Type a message..."
                rows={1}
                className="max-h-[120px] min-h-[36px] flex-1 resize-none rounded-md border border-gray-300 px-3 py-2 text-sm focus:border-blue-500 focus:outline-none focus:ring-1 focus:ring-blue-500"
              />
              <button
                onClick={handleSend}
                disabled={!input.trim()}
                className={clsx(
                  "flex h-9 w-9 shrink-0 items-center justify-center rounded-md transition-colors",
                  input.trim()
                    ? "bg-blue-600 text-white hover:bg-blue-700"
                    : "bg-gray-100 text-gray-400",
                )}
              >
                <Send className="h-4 w-4" />
              </button>
            </div>
          )}
        </div>
      </div>
    </>
  );
}
