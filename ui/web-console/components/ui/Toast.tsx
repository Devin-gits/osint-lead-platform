"use client";

import { cn } from "@/lib/utils/cn";

export interface ToastProps {
  message: string;
  visible?: boolean;
  variant?: "info" | "success" | "warning" | "danger";
}

export function Toast({ message, visible = true, variant = "info" }: ToastProps) {
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
        "rounded-md border px-4 py-3 text-sm shadow-lg",
        variants[variant]
      )}
      role="status"
    >
      {message}
    </div>
  );
}
