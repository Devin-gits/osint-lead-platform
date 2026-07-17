"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { ArrowLeft, Play, RefreshCw, AlertTriangle, AlertCircle } from "lucide-react";
import Link from "next/link";
import { useLead, useRunLeadModules, useModules } from "@/lib/api/hooks";
import { useToast } from "@/components/providers/ToastProvider";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Tabs } from "@/components/ui/Tabs";
import { AuditLogPanel } from "@/components/ui/AuditLogPanel";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip } from "@/components/ui/StatusChip";
import { EmptyState } from "@/components/ui/EmptyState";
import { cn } from "@/lib/utils/cn";
import {
  DomainIntelResult,
  ModuleName,
  SocialFootprintResult,
} from "@/lib/api/types";

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
  const { addToast } = useToast();
  const [running, setRunning] = useState<ModuleName | null>(null);

  const handleRun = async (module: ModuleName) => {
    setRunning(module);
    try {
      const updated = await run.mutateAsync({ id, body: { modules: [module] } });
      const resultKey = module.replace(/-/g, "_");
      const result = ((updated as unknown) as Record<string, unknown>)[resultKey] as { status?: string } | undefined;
      const status = result?.status || "unknown";
      const variant = status === "ok" ? "success" : status === "skipped" ? "warning" : "danger";
      addToast(`${module} finished with status: ${status}`, variant);
      refetch();
    } catch (err) {
      addToast(
        err instanceof Error ? err.message : `Failed to run ${module}`,
        "danger"
      );
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
            <Field
              label="Permission ref"
              value={
                lead.permission_ref ? (
                  lead.permission_ref
                ) : (
                  <Badge variant="warning" className="gap-1">
                    <AlertTriangle className="h-3 w-3" />
                    Missing
                  </Badge>
                )
              }
            />
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
          {(result.reason as string) || "This module did not run."}
        </p>
        {module !== "email-validate" && module !== "phone-validate" && (
          <Button size="sm" variant="ghost" onClick={onRun} disabled={isRunning}>
            {isRunning ? "Running…" : "Run anyway"}
          </Button>
        )}
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

      {module === "domain-intel" ? (
        <DomainResultPanel result={result as unknown as DomainIntelResult} />
      ) : module === "social-footprint" ? (
        <SocialResultPanel result={result as unknown as SocialFootprintResult} />
      ) : (
        <GenericResultGrid result={result} />
      )}
    </div>
  );
}

function DomainResultPanel({ result }: { result: DomainIntelResult }) {
  const wc = result.web_check;
  const hv = result.harvester;

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <ResultCard label="Resolvable" value={wc?.resolvable === true ? "yes" : wc?.resolvable === false ? "no" : "—"} />
        <ResultCard label="DNS status" value={<StatusChip status={wc?.status || "unknown"} className="text-xs" />} />
        <ResultCard label="SSL status" value={<StatusChip status={wc?.ssl?.valid ? "ok" : wc?.ssl ? "unknown" : "not_run"} className="text-xs" />} />
        <ResultCard label="Harvester status" value={<StatusChip status={hv?.status || "not_run"} className="text-xs" />} />
      </div>

      {wc?.dns && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">DNS records</h4>
          <div className="grid gap-2 text-sm sm:grid-cols-2">
            {wc.dns.a && wc.dns.a.length > 0 && <RecordList label="A" items={wc.dns.a} />}
            {wc.dns.mx && wc.dns.mx.length > 0 && <RecordList label="MX" items={wc.dns.mx} />}
            {wc.dns.ns && wc.dns.ns.length > 0 && <RecordList label="NS" items={wc.dns.ns} />}
            {wc.dns.txt && wc.dns.txt.length > 0 && (
              <div>
                <span className="text-xs text-foreground-muted">TXT</span>
                <div className="mt-1 text-xs text-foreground-secondary">{wc.dns.txt.length} record(s)</div>
              </div>
            )}
          </div>
        </div>
      )}

      {wc?.ssl && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">SSL/TLS</h4>
          <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
            <ResultCard label="Valid" value={wc.ssl.valid ? "yes" : "no"} />
            {wc.ssl.days_until_expiry !== undefined && <ResultCard label="Days until expiry" value={wc.ssl.days_until_expiry} />}
            {wc.ssl.protocol && <ResultCard label="Protocol" value={wc.ssl.protocol} />}
            {wc.ssl.subject && <ResultCard label="Subject" value={wc.ssl.subject} />}
            {wc.ssl.issuer && <ResultCard label="Issuer" value={wc.ssl.issuer} />}
            {wc.ssl.sans && wc.ssl.sans.length > 0 && <RecordList label="SANs" items={wc.ssl.sans} />}
          </div>
        </div>
      )}

      {wc?.http && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">HTTP</h4>
          <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
            {wc.http.status_code !== undefined && <ResultCard label="Status code" value={wc.http.status_code} />}
            {wc.http.server && <ResultCard label="Server" value={wc.http.server} />}
          </div>
        </div>
      )}

      {wc?.whois && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">WHOIS</h4>
          <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-3">
            {wc.whois.registrar && <ResultCard label="Registrar" value={wc.whois.registrar} />}
            {wc.whois.domain_age_days !== undefined && <ResultCard label="Domain age (days)" value={wc.whois.domain_age_days} />}
            {wc.whois.created_date && <ResultCard label="Created" value={new Date(wc.whois.created_date).toLocaleDateString()} />}
          </div>
        </div>
      )}

      {hv && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">theHarvester</h4>
          <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-4">
            <ResultCard label="Status" value={<StatusChip status={hv.status} className="text-xs" />} />
            {hv.emails && <ResultCard label="Emails found" value={hv.emails.length} />}
            {hv.hosts && <ResultCard label="Hosts found" value={hv.hosts.length} />}
            {hv.ips && <ResultCard label="IPs found" value={hv.ips.length} />}
            {hv.error && <ResultCard label="Error" value={hv.error} className="col-span-full text-danger" />}
          </div>
        </div>
      )}

      <RawJsonView data={result} />
    </div>
  );
}

function SocialResultPanel({ result }: { result: SocialFootprintResult }) {
  const handles = result.handles || [];

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <ResultCard label="Active signals" value={result.active_signals ?? 0} />
        <ResultCard label="Confidence" value={`${Math.round((result.confidence ?? 0) * 100)}%`} />
        <ResultCard label="Handles checked" value={result.handles_checked?.length ?? 0} />
        <ResultCard label="Source tool" value={result.source_tool || "—"} />
      </div>

      {result.rate_limit_note && (
        <p className="text-xs text-foreground-muted">{result.rate_limit_note}</p>
      )}

      <div className="space-y-3">
        {handles.map((h) => (
          <div key={h.handle} className="rounded-md border border-border bg-surface p-4">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <div className="font-medium text-foreground">{h.handle}</div>
                <div className="text-xs text-foreground-muted">origin: {h.origin}</div>
              </div>
              <StatusChip status={h.status} className="text-xs" />
            </div>

            {h.error && <p className="mt-2 text-sm text-danger">{h.error}</p>}

            <div className="mt-3 flex flex-wrap gap-2">
              {h.platforms.map((p) => (
                <Badge
                  key={p.platform}
                  variant={p.status === "claimed" ? "success" : p.status === "available" ? "outline" : "muted"}
                  className="text-xs"
                >
                  {p.platform}: {p.status}
                </Badge>
              ))}
            </div>

            {h.claimed_count > 0 && (
              <div className="mt-2 text-xs text-foreground-muted">
                {h.claimed_count} claimed platform(s)
              </div>
            )}
          </div>
        ))}
      </div>

      <RawJsonView data={result} />
    </div>
  );
}

function GenericResultGrid({ result }: { result: Record<string, unknown> }) {
  return (
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
  );
}

function ResultCard({
  label,
  value,
  className,
}: {
  label: string;
  value: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("rounded-md border border-border bg-surface p-3", className)}>
      <div className="text-xs text-foreground-muted">{label}</div>
      <div className="mt-1 text-sm font-medium text-foreground">{value}</div>
    </div>
  );
}

function RecordList({ label, items }: { label: string; items: string[] }) {
  return (
    <div>
      <span className="text-xs text-foreground-muted">{label}</span>
      <ul className="mt-1 list-inside list-disc text-xs text-foreground-secondary">
        {items.slice(0, 4).map((item) => (
          <li key={item} className="break-all">{item}</li>
        ))}
        {items.length > 4 && <li>+{items.length - 4} more</li>}
      </ul>
    </div>
  );
}

function RawJsonView({ data }: { data: unknown }) {
  const [open, setOpen] = useState(false);
  return (
    <div className="rounded-md border border-border bg-surface p-3">
      <button
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between text-xs font-medium text-foreground-secondary hover:text-foreground"
      >
        <span>Raw JSON</span>
        <span>{open ? "−" : "+"}</span>
      </button>
      {open && (
        <pre className="mt-2 max-h-64 overflow-auto rounded-md bg-surface-elevated p-3 text-xs text-foreground-secondary">
          {JSON.stringify(data, null, 2)}
        </pre>
      )}
    </div>
  );
}
