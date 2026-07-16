"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { ArrowLeft, Play, RefreshCw } from "lucide-react";
import Link from "next/link";
import { useLead, useRunLeadModules, useModules } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Tabs } from "@/components/ui/Tabs";
import { AuditLogPanel } from "@/components/ui/AuditLogPanel";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip } from "@/components/ui/StatusChip";
import { EmptyState } from "@/components/ui/EmptyState";
import { AlertCircle } from "lucide-react";
import { ModuleName } from "@/lib/api/types";

const moduleOrder: { key: string; label: string; module: ModuleName }[] = [
  { key: "email_validate", label: "Email", module: "email-validate" },
  { key: "phone_validate", label: "Phone", module: "phone-validate" },
  { key: "domain_intel", label: "Domain", module: "domain-intel" },
  { key: "social_footprint", label: "Social", module: "social-footprint" },
];

export default function LeadDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: lead, isLoading, error, refetch } = useLead(id);
  const { data: modules } = useModules();
  const run = useRunLeadModules();
  const [running, setRunning] = useState<ModuleName | null>(null);

  const handleRun = async (module: ModuleName) => {
    setRunning(module);
    try {
      await run.mutateAsync({ id, body: { modules: [module] } });
      refetch();
    } finally {
      setRunning(null);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (error || !lead) {
    return (
      <div className="space-y-6">
        <PageHeader title="Lead not found" />
        <Card className="p-6">
          <EmptyState
            icon={AlertCircle}
            title="Failed to load lead"
            description={error?.message || "This lead does not exist."}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={lead.name || lead.email || lead.id}
        description={`${lead.company || "No company"} • ${lead.email || "no email"}`}
      >
        <Link href="/leads">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-1.5 h-4 w-4" />
            Back to leads
          </Button>
        </Link>
      </PageHeader>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Stage" value={<Badge className="capitalize">{lead.stage.replace("_", " ")}</Badge>} />
            <Field label="Risk level" value={<RiskBadge level={lead.risk_level} />} />
            {lead.risk_score !== undefined && <Field label="Risk score" value={lead.risk_score} />}
            <Field label="Email" value={lead.email || "—"} />
            <Field label="Phone" value={lead.phone || "—"} />
            <Field label="Company" value={lead.company || "—"} />
            <Field label="Domain" value={lead.domain || "—"} />
            <Field label="Source ID" value={lead.source_id || "—"} />
            <Field label="Permission ref" value={lead.permission_ref || "—"} />
            <Field label="Updated" value={new Date(lead.updated_at).toLocaleString()} />
          </div>
        </Card>

        <Card>
          <h3 className="mb-3 text-sm font-semibold text-foreground">Run module</h3>
          <div className="space-y-2">
            {moduleOrder.map(({ label, module }) => {
              const mod = modules?.find((m) => m.name === module);
              const isWired = mod?.dev_status === "available";
              return (
                <Button
                  key={module}
                  size="sm"
                  variant="secondary"
                  className="w-full justify-between"
                  disabled={!isWired || running !== null || run.isPending}
                  onClick={() => handleRun(module)}
                >
                  <span className="flex items-center gap-2">
                    <Play className="h-3.5 w-3.5" />
                    {label}
                  </span>
                  {!isWired && <span className="text-[10px] opacity-70">not wired</span>}
                </Button>
              );
            })}
          </div>
          {run.error && (
            <div className="mt-3 text-sm text-danger">{run.error.message}</div>
          )}
        </Card>
      </div>

      <Card>
        <Tabs
          defaultTab="email_validate"
          tabs={moduleOrder.map(({ key, label, module }) => {
            const result = ((lead as unknown) as Record<string, unknown>)[key] as Record<string, unknown> | undefined;
            return {
              id: key,
              label: `${label} ${result ? `(${result.status || "n/a"})` : ""}`,
              content: (
                <ModuleResultPanel
                  module={module}
                  result={result}
                  onRun={() => handleRun(module)}
                  isRunning={running === module}
                />
              ),
            };
          })}
        />
      </Card>

      <Card>
        <div className="mb-3 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-foreground">Audit trail</h3>
          <Button size="sm" variant="ghost" onClick={() => refetch()} disabled={isLoading}>
            <RefreshCw className="mr-1.5 h-4 w-4" />
            Refresh
          </Button>
        </div>
        <AuditLogPanel events={lead.audit_events || []} />
      </Card>
    </div>
  );
}

function Field({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div>
      <div className="text-xs text-foreground-muted">{label}</div>
      <div className="text-sm font-medium text-foreground">{value}</div>
    </div>
  );
}

function RiskBadge({ level }: { level: string }) {
  const variant =
    level === "low" ? "success" : level === "medium" ? "warning" : level === "high" ? "danger" : "outline";
  return <Badge variant={variant}>{level}</Badge>;
}

function ModuleResultPanel({
  module,
  result,
  onRun,
  isRunning,
}: {
  module: ModuleName;
  result?: Record<string, unknown>;
  onRun: () => void;
  isRunning: boolean;
}) {
  const status = (result?.status as string) || "not_run";

  if (!result) {
    return (
      <EmptyState
        icon={AlertCircle}
        title="Not run yet"
        description={`Run ${module} to see results.`}
        className="py-8"
      />
    );
  }

  if (status === "skipped") {
    return (
      <div className="space-y-4 py-4">
        <StatusChip status="skipped" />
        <p className="text-sm text-foreground-muted">
          {result.reason as string || "This module is not wired in control-plane v1."}
        </p>
      </div>
    );
  }

  return (
    <div className="space-y-4 py-4">
      <div className="flex items-center justify-between">
        <StatusChip status={status as "ok" | "unknown" | "skipped" | "pending" | "not_run"} />
        {module !== "email-validate" && module !== "phone-validate" && (
          <Button size="sm" variant="ghost" onClick={onRun} disabled={isRunning}>
            {isRunning ? "Running…" : "Run anyway"}
          </Button>
        )}
      </div>

      <div className="grid gap-3 sm:grid-cols-2">
        {Object.entries(result).map(([key, value]) => {
          if (key === "status" || value === undefined || value === null) return null;
          return (
            <div key={key} className="rounded-md border border-border bg-surface p-3">
              <div className="text-xs text-foreground-muted capitalize">{key.replace(/_/g, " ")}</div>
              <div className="mt-1 text-sm text-foreground">
                {typeof value === "boolean" ? (value ? "yes" : "no") : typeof value === "object" ? JSON.stringify(value) : String(value)}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
