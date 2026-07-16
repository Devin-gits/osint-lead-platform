"use client";

import { cn } from "@/lib/utils/cn";
import { X } from "lucide-react";

export interface ToastProps {
  message: string;
  visible?: boolean;
  variant?: "info" | "success" | "warning" | "danger";
  onClose?: () => void;
}

export function Toast({ message, visible = true, variant = "info", onClose }: ToastProps) {
  const variants = {
    info: "border-primary/30 bg-surface-elevated text-primary",
    success: "border-success/30 bg-surface-elevated text-success",
    warning: "border-warning/30 bg-surface-elevated text-warning",
    danger: "border-danger/30 bg-surface-elevated text-danger",
  };

  if (!visible) return null;

  return (
    <div
      className={cn(
        "flex items-start justify-between gap-3 rounded-md border px-4 py-3 text-sm shadow-lg",
        variants[variant]
      )}
      role="status"
    >
      <span className="flex-1">{message}</span>
      {onClose && (
        <button
          onClick={onClose}
          aria-label="Dismiss"
          className="mt-0.5 rounded p-0.5 hover:bg-black/5 focus:outline-none focus:ring-1 focus:ring-current"
        >
          <X className="h-4 w-4" aria-hidden="true" />
        </button>
      )}
    </div>
  );
}
