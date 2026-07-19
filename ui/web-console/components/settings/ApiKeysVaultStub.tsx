import { Badge } from "@/components/ui/Badge";

export function ApiKeysVaultStub() {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">API keys vault</h3>
        <Badge variant="muted">Not configured</Badge>
      </div>
      <p className="text-xs text-foreground-muted">
        Secure storage for third-party API credentials (numverify, breach APIs).
        Keys will be managed server-side via the control-plane — never stored in the browser.
      </p>
      <div className="rounded-md border border-border bg-surface-elevated p-3 text-xs text-foreground-muted">
        No secrets are stored client-side. This section will connect to a backend
        vault when available.
      </div>
    </div>
  );
}
