"use client";

import { useState, useCallback, type MouseEvent } from "react";
import { Copy, Check } from "lucide-react";
import { toastSuccess, toastError } from "@/lib/toast";

interface CopyableIDProps {
  id: string | undefined | null;
  label?: string;
  truncate?: number;
}

export default function CopyableID({ id, label, truncate = 8 }: CopyableIDProps) {
  const [copied, setCopied] = useState(false);

  const handleCopy = useCallback(
    async (e: MouseEvent<HTMLButtonElement>) => {
      e.stopPropagation();
      e.preventDefault();
      if (!id) return;
      try {
        await navigator.clipboard.writeText(id);
        setCopied(true);
        toastSuccess(`${label ?? "ID"} copied`);
        setTimeout(() => setCopied(false), 1500);
      } catch (err) {
        toastError(err);
      }
    },
    [id, label],
  );

  if (!id) {
    return <span className="text-gray-400">-</span>;
  }

  const short = id.length > truncate ? `${id.slice(0, truncate)}...` : id;

  return (
    <button
      type="button"
      onClick={handleCopy}
      title={id}
      className="group inline-flex items-center gap-1.5 rounded px-1 py-0.5 font-mono text-xs text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-900"
    >
      <span>{short}</span>
      {copied ? (
        <Check className="h-3 w-3 text-green-600" />
      ) : (
        <Copy className="h-3 w-3 text-gray-400 group-hover:text-blue-500" />
      )}
    </button>
  );
}
