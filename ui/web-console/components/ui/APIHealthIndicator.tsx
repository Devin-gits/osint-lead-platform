"use client";

import { useApiHealth } from "@/lib/api/hooks";
import { cn } from "@/lib/utils/cn";
import { RefreshCw, AlertCircle, CheckCircle2, Loader2 } from "lucide-react";

export function APIHealthIndicator({ className }: { className?: string }) {
  const { isLoading, isError, isSuccess, refetch, dataUpdatedAt, error } = useApiHealth();

  const lastChecked = dataUpdatedAt
    ? new Date(dataUpdatedAt).toLocaleTimeString()
    : undefined;

  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-3 rounded-md border border-border bg-surface px-4 py-3 text-sm",
        className
      )}
      aria-live="polite"
    >
      {isLoading && !isSuccess && (
        <>
          <Loader2 className="h-4 w-4 shrink-0 animate-spin text-foreground-muted" aria-hidden="true" />
          <span className="text-foreground-secondary">
            Checking control-plane API…
          </span>
        </>
      )}

      {isSuccess && (
        <>
          <CheckCircle2 className="h-4 w-4 shrink-0 text-success" aria-hidden="true" />
          <span className="text-foreground-secondary">
            API reachable
            {lastChecked && <span className="ml-1 text-foreground-muted">(last checked {lastChecked})</span>}
          </span>
        </>
      )}

      {isError && (
        <>
          <AlertCircle className="h-4 w-4 shrink-0 text-danger" aria-hidden="true" />
          <span className="text-foreground-secondary">
            Cannot reach control-plane API
            {error instanceof Error && (
              <span className="ml-1 text-foreground-muted">— {error.message}</span>
            )}
          </span>
        </>
      )}

      <button
        type="button"
        onClick={() => refetch()}
        disabled={isLoading}
        className="ml-auto inline-flex items-center gap-1.5 rounded-md px-2 py-1 text-xs font-medium text-foreground-secondary hover:bg-surface-elevated hover:text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50"
        aria-label="Retry API reachability check"
      >
        <RefreshCw className={cn("h-3.5 w-3.5", isLoading && "animate-spin")} aria-hidden="true" />
        Retry
      </button>
    </div>
  );
}
