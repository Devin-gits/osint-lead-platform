"use client";

import { Info, Terminal } from "lucide-react";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { EnvironmentSetting } from "@/components/settings/EnvironmentSetting";
import { RoleSelector } from "@/components/settings/RoleSelector";
import { CrmConnectorStub } from "@/components/settings/CrmConnectorStub";
import { SsoOidStub } from "@/components/settings/SsoOidStub";
import { ApiKeysVaultStub } from "@/components/settings/ApiKeysVaultStub";
import { RetentionPolicyStub } from "@/components/settings/RetentionPolicyStub";

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <PageHeader
        title="Settings"
        description="Environment and future admin controls (stubs until auth/CRM exist)."
      />

      <div className="flex items-start gap-2 rounded-md border border-warning/20 bg-warning/5 p-3 text-sm text-foreground-secondary">
        <Info className="mt-0.5 h-4 w-4 shrink-0 text-warning" />
        <span>
          Pre-production — values are local UI stubs unless noted.
          No secrets are stored in the browser.
        </span>
      </div>

      <div className="grid gap-6 lg:grid-cols-2">
        {/* Environment + Role */}
        <Card>
          <EnvironmentSetting />
        </Card>
        <Card>
          <RoleSelector />
        </Card>

        {/* Connector stubs */}
        <Card>
          <CrmConnectorStub />
        </Card>
        <Card>
          <SsoOidStub />
        </Card>
        <Card>
          <ApiKeysVaultStub />
        </Card>
        <Card>
          <RetentionPolicyStub />
        </Card>
      </div>

      {/* Local ops note */}
      <Card>
        <div className="flex items-start gap-3">
          <Terminal className="mt-0.5 h-5 w-5 shrink-0 text-foreground-muted" />
          <div className="space-y-2">
            <h3 className="text-sm font-semibold text-foreground">Local development</h3>
            <div className="space-y-1 text-xs text-foreground-muted">
              <p>
                <span className="font-medium text-foreground">UI:</span>{" "}
                <code className="rounded bg-surface-elevated px-1 py-0.5">
                  npx next dev -H 127.0.0.1 -p 3000
                </code>{" "}
                then open <span className="font-medium">http://localhost:3000</span>
              </p>
              <p>
                <span className="font-medium text-foreground">API:</span>{" "}
                Control-plane defaults to loopback (127.0.0.1:8080).
                Override with <code className="rounded bg-surface-elevated px-1 py-0.5">LISTEN_HOST=0.0.0.0</code> for deployment.
              </p>
              <p>
                <span className="font-medium text-foreground">CORS:</span>{" "}
                Default origin is <code className="rounded bg-surface-elevated px-1 py-0.5">http://localhost:3000</code>.
                Browsers treat localhost and 127.0.0.1 as different origins — always open the UI via localhost.
              </p>
            </div>
          </div>
        </div>
      </Card>
    </div>
  );
}
