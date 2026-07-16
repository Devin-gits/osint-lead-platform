"use client";

import { CheckCircle2, AlertCircle, Loader2 } from "lucide-react";
import { API_BASE } from "@/lib/api/client";
import { useApiHealth } from "@/lib/api/hooks";

export function EnvironmentSetting() {
  const { isLoading, isError, isSuccess } = useApiHealth();

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-foreground">Environment</h3>
      <div className="text-sm text-foreground-secondary">
        <div>API base URL</div>
        <div className="font-medium text-foreground">{API_BASE}</div>
      </div>
      <div className="flex items-center gap-2 text-sm">
        {isLoading && <Loader2 className="h-4 w-4 animate-spin text-foreground-muted" />}
        {isSuccess && <CheckCircle2 className="h-4 w-4 text-success" />}
        {isError && <AlertCircle className="h-4 w-4 text-danger" />}
        <span className="text-foreground-secondary">
          {isLoading ? "Checking…" : isError ? "API unreachable" : "API reachable"}
        </span>
      </div>
    </div>
  );
}
