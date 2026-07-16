"use client";

import { AlertTriangle, CheckCircle2, Loader2 } from "lucide-react";
import { API_BASE } from "@/lib/api/client";
import { useApiHealth } from "@/lib/api/hooks";

export function EnvironmentBanner() {
  const { isLoading, isError } = useApiHealth();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center gap-2 border-b border-border bg-surface px-4 py-2 text-xs text-foreground-muted">
        <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
        <span>Checking API at {API_BASE}…</span>
      </div>
    );
  }

  if (isError) {
    return (
      <div
        className="flex items-center justify-center gap-2 border-b border-danger/20 bg-danger/10 px-4 py-2 text-xs font-medium text-danger"
        role="status"
      >
        <AlertTriangle className="h-4 w-4" aria-hidden="true" />
        <span>API unreachable — start services/control-plane ({API_BASE})</span>
      </div>
    );
  }

  return (
    <div
      className="flex items-center justify-center gap-2 border-b border-success/20 bg-success/10 px-4 py-2 text-xs font-medium text-success"
      role="status"
    >
      <CheckCircle2 className="h-4 w-4" aria-hidden="true" />
      <span>Live API — {API_BASE}</span>
    </div>
  );
}
