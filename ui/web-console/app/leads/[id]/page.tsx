"use client";

import { useState } from "react";
import { useParams } from "next/navigation";
import { ArrowLeft, Play, RefreshCw, AlertTriangle, AlertCircle, ExternalLink, Download, TrendingUp, TrendingDown } from "lucide-react";
import Link from "next/link";
import { useLead, useRunLeadModules, useModules, useLeadReadiness, usePromoteLead, useDemoteLead, useExportLead } from "@/lib/api/hooks";
import { useToast } from "@/components/providers/ToastProvider";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Tabs } from "@/components/ui/Tabs";
import { AuditLogPanel } from "@/components/ui/AuditLogPanel";
import { Skeleton } from "@/components/ui/Skeleton";
import { StatusChip, type StatusChipStatus } from "@/components/ui/StatusChip";
import { EmptyState } from "@/components/ui/EmptyState";
import { cn } from "@/lib/utils/cn";
import {
  CompanyEnrichResult,
  DomainIntelResult,
  ExtractionResult,
  ModuleName,
  ReadinessCheck,
  ReadinessReport,
  SocialFootprintResult,
} from "@/lib/api/types";

const moduleOrder: { key: string; label: string; module: ModuleName }[] = [
  { key: "email_validate", label: "Email", module: "email-validate" },
  { key: "phone_validate", label: "Phone", module: "phone-validate" },
  { key: "domain_intel", label: "Domain", module: "domain-intel" },
  { key: "social_footprint", label: "Social", module: "social-footprint" },
  { key: "extraction", label: "Extraction", module: "extraction" },
  { key: "company_enrich", label: "Company", module: "company-enrich" },
];

type ModuleInfoLike = { dev_status?: string } | undefined;

function moduleRunState(
  module: ModuleName,
  mod: ModuleInfoLike,
  lead?: { url?: string; company?: string; domain?: string; permission_ref?: string }
): { canRun: boolean; disabledReason: string | null } {
  const isWired = mod?.dev_status === "available";
  if (!isWired) return { canRun: false, disabledReason: "not wired" };
  if (module === "extraction") {
    if (!lead?.url) return { canRun: false, disabledReason: "needs url" };
    if (!lead?.permission_ref) return { canRun: false, disabledReason: "needs permission ref" };
  }
  if (module === "company-enrich") {
    if (!lead?.permission_ref) return { canRun: false, disabledReason: "needs permission ref" };
    if (!lead?.domain && !lead?.company && !lead?.url) {
      return { canRun: false, disabledReason: "needs domain, company, or url" };
    }
  }
  return { canRun: true, disabledReason: null };
}

export default function LeadDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: lead, isLoading, error, refetch } = useLead(id);
  const { data: readiness, isLoading: readinessLoading } = useLeadReadiness(id);
  const { data: modules } = useModules();
  const run = useRunLeadModules();
  const promote = usePromoteLead();
  const demote = useDemoteLead();
  const exportLead = useExportLead();
  const { addToast } = useToast();
  const [running, setRunning] = useState<ModuleName | null>(null);
  const [transitioning, setTransitioning] = useState(false);

  const handlePromote = async () => {
    setTransitioning(true);
    try {
      await promote.mutateAsync({ id, body: { target: "crm_ready" } });
      addToast("Lead promoted to CRM-ready", "success");
      refetch();
    } catch (err) {
      if (err && typeof err === "object" && "status" in err && err.status === 409) {
        const report = (err as { data?: ReadinessReport }).data;
        const failed = report?.checks.filter((c) => !c.pass).map((c) => c.name).join(", ") || "unknown";
        addToast(`Not ready: ${failed}`, "warning");
      } else {
        addToast(err instanceof Error ? err.message : "Failed to promote lead", "danger");
      }
    } finally {
      setTransitioning(false);
    }
  };

  const handleDemote = async () => {
    if (lead?.stage !== "crm_ready") return;
    setTransitioning(true);
    try {
      await demote.mutateAsync({ id, body: { target: "validated" } });
      addToast("Lead demoted to validated", "success");
      refetch();
    } catch (err) {
      addToast(err instanceof Error ? err.message : "Failed to demote lead", "danger");
    } finally {
      setTransitioning(false);
    }
  };

  const handleExport = async () => {
    try {
      const data = await exportLead.mutateAsync(id);
      const blob = new Blob([JSON.stringify(data, null, 2)], { type: "application/json" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `lead-${id}-export.json`;
      a.click();
      URL.revokeObjectURL(url);
      addToast("CRM export stub downloaded", "success");
    } catch (err) {
      if (err && typeof err === "object" && "status" in err && err.status === 409) {
        const report = (err as { data?: ReadinessReport }).data;
        const failed = report?.checks.filter((c) => !c.pass).map((c) => c.name).join(", ") || "unknown";
        addToast(`Export blocked: ${failed}`, "warning");
      } else {
        addToast(err instanceof Error ? err.message : "Failed to export lead", "danger");
      }
    }
  };

  const handleRun = async (module: ModuleName) => {
    setRunning(module);
    try {
      const updated = await run.mutateAsync({
        id,
        body: {
          modules: [module],
          ...(lead?.permission_ref ? { permission_ref: lead.permission_ref } : {}),
        },
      });
      const resultKey = module.replace(/-/g, "_");
      const result = ((updated as unknown) as Record<string, unknown>)[resultKey] as { status?: string } | undefined;
      const status = result?.status || "unknown";
      const variant =
        status === "ok" ? "success" : status === "skipped" || status === "partial" ? "warning" : "danger";
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
        <Link
          href="/leads"
          className="inline-flex items-center rounded-md bg-transparent px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-surface-elevated focus:outline-none focus:ring-2 focus:ring-primary/50"
        >
          <ArrowLeft className="mr-1.5 h-4 w-4" />
          Back to leads
        </Link>
      </PageHeader>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2">
          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Stage" value={<Badge className="capitalize">{lead.stage.replace(/_/g, " ")}</Badge>} />
            <Field label="Risk level" value={<RiskBadge level={lead.risk_level} />} />
            {lead.risk_score !== undefined && <Field label="Risk score" value={lead.risk_score} />}
            <Field label="Email" value={lead.email || "—"} />
            <Field label="Phone" value={lead.phone || "—"} />
            <Field label="Company" value={lead.company || "—"} />
            <Field label="Domain" value={lead.domain || "—"} />
            <Field label="URL" value={lead.url || "—"} />
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
          {!lead.permission_ref && (
            <div className="mb-3 flex items-start gap-2 rounded-md border border-warning/20 bg-warning/10 p-2 text-xs text-warning">
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              <span>No permission reference — extraction and other module runs will be skipped.</span>
            </div>
          )}
          {!lead.url && (
            <div className="mb-3 flex items-start gap-2 rounded-md border border-warning/20 bg-warning/10 p-2 text-xs text-warning">
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              <span>No URL — extraction requires a public URL.</span>
            </div>
          )}
          {!lead.domain && !lead.company && !lead.url && (
            <div className="mb-3 flex items-start gap-2 rounded-md border border-warning/20 bg-warning/10 p-2 text-xs text-warning">
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              <span>No domain, company, or URL — company enrichment needs one of them.</span>
            </div>
          )}
          <div className="space-y-2">
            {moduleOrder.map(({ label, module }) => {
              const mod = modules?.find((m) => m.name === module);
              const { canRun, disabledReason } = moduleRunState(module, mod, lead);
              return (
                <Button
                  key={module}
                  size="sm"
                  variant="secondary"
                  className="w-full justify-between"
                  disabled={!canRun || running !== null || run.isPending}
                  onClick={() => handleRun(module)}
                >
                  <span className="flex items-center gap-2">
                    <Play className="h-3.5 w-3.5" />
                    {label}
                  </span>
                  {disabledReason && <span className="text-[10px] opacity-70">{disabledReason}</span>}
                </Button>
              );
            })}
          </div>
          {run.error && (
            <div className="mt-3 text-sm text-danger">{run.error.message}</div>
          )}
        </Card>
      </div>

      <CRMReadinessSection
        stage={lead.stage}
        readiness={readiness}
        loading={readinessLoading}
        transitioning={transitioning}
        onPromote={handlePromote}
        onDemote={handleDemote}
        onExport={handleExport}
      />

      <Card>
        <Tabs
          defaultTab="email_validate"
          tabs={moduleOrder.map(({ key, label, module }) => {
            const result = ((lead as unknown) as Record<string, unknown>)[key] as Record<string, unknown> | undefined;
            const mod = modules?.find((m) => m.name === module);
            const { canRun, disabledReason } = moduleRunState(module, mod, lead);
            return {
              id: key,
              label: `${label} ${result ? `(${result.status || "n/a"})` : ""}`,
              content: (
                <ModuleResultPanel
                  module={module}
                  result={result}
                  onRun={() => handleRun(module)}
                  isRunning={running === module}
                  canRun={canRun}
                  disabledReason={disabledReason}
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
  canRun,
  disabledReason,
}: {
  module: ModuleName;
  result?: Record<string, unknown>;
  onRun: () => void;
  isRunning: boolean;
  canRun: boolean;
  disabledReason: string | null;
}) {
  const status = (result?.status as string) || "not_run";
  const runLabel =
    module === "extraction" ? "Run extraction" :
    module === "company-enrich" ? "Run company enrich" :
    "Run anyway";

  const RunButton = (
    <Button size="sm" variant="ghost" onClick={onRun} disabled={isRunning || !canRun}>
      {isRunning ? "Running…" : runLabel}
    </Button>
  );

  if (!result) {
    return (
      <div className="space-y-4 py-8">
        <EmptyState
          icon={AlertCircle}
          title="Not run yet"
          description={`Run ${module} to see results.`}
        />
        {disabledReason && (
          <p className="text-xs text-warning">{disabledReason}</p>
        )}
        {module !== "email-validate" && module !== "phone-validate" && RunButton}
      </div>
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
          <>
            {RunButton}
            {disabledReason && (
              <p className="text-xs text-warning">{disabledReason}</p>
            )}
          </>
        )}
      </div>
    );
  }

  return (
    <div className="space-y-4 py-4">
      <div className="flex items-center justify-between">
        <StatusChip status={status as StatusChipStatus} />
        {module !== "email-validate" && module !== "phone-validate" && (
          <>
            {RunButton}
          </>
        )}
      </div>

      {module === "domain-intel" ? (
        <DomainResultPanel result={result as unknown as DomainIntelResult} />
      ) : module === "social-footprint" ? (
        <SocialResultPanel result={result as unknown as SocialFootprintResult} />
      ) : module === "extraction" ? (
        <ExtractionResultPanel result={result as unknown as ExtractionResult} />
      ) : module === "company-enrich" ? (
        <CompanyResultPanel result={result as unknown as CompanyEnrichResult} />
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

function ExtractionResultPanel({ result }: { result: ExtractionResult }) {
  const fields = result.fields;
  const provenance = result.provenance ?? [];
  const [rawOpen, setRawOpen] = useState(false);

  const showConfidence = result.status === "ok" || result.status === "partial";
  const confidenceValue =
    showConfidence && result.confidence !== undefined && result.confidence !== null
      ? `${Math.round(result.confidence * 100)}%`
      : "—";

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <ResultCard label="Source tool" value={result.source_tool || "—"} />
        <ResultCard label="Confidence" value={confidenceValue} />
        <ResultCard label="HTTP status" value={result.metadata?.http_status ?? "—"} />
        <ResultCard label="Final URL" value={result.final_url || result.url || "—"} />
      </div>

      {result.error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-3 text-sm text-danger">
          {result.error}
        </div>
      )}

      {fields && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">Extracted fields</h4>
          <div className="grid gap-2 text-sm sm:grid-cols-2">
            {fields.company_name && <ResultCard label="Company name" value={fields.company_name} />}
            {fields.title && <ResultCard label="Title" value={fields.title} />}
            {fields.description && (
              <div className="col-span-full">
                <span className="text-xs text-foreground-muted">Description</span>
                <p className="mt-1 text-sm text-foreground">{fields.description}</p>
              </div>
            )}
            {fields.emails && fields.emails.length > 0 && (
              <RecordList label="Emails" items={fields.emails} />
            )}
            {fields.phones && fields.phones.length > 0 && (
              <RecordList label="Phones" items={fields.phones} />
            )}
            {fields.social_links && fields.social_links.length > 0 && (
              <RecordList label="Social links" items={fields.social_links} />
            )}
            {fields.contact_urls && fields.contact_urls.length > 0 && (
              <RecordList label="Contact URLs" items={fields.contact_urls} />
            )}
            {fields.addresses && fields.addresses.length > 0 && (
              <RecordList label="Addresses" items={fields.addresses} />
            )}
          </div>
        </div>
      )}

      {provenance.length > 0 && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">Provenance</h4>
          <ul className="space-y-2">
            {provenance.slice(0, 20).map((p, idx) => (
              <li key={idx} className="text-xs text-foreground-secondary">
                <span className="font-medium text-foreground">{p.field}</span>
                {": "}
                {p.value}
                {" "}
                <span className="text-foreground-muted">({p.method} @ {p.source_url} — {new Date(p.timestamp).toLocaleString()})</span>
              </li>
            ))}
            {provenance.length > 20 && <li className="text-xs text-foreground-muted">+{provenance.length - 20} more</li>}
          </ul>
        </div>
      )}

      {result.raw_markdown && (
        <div className="rounded-md border border-border bg-surface p-4">
          <button
            onClick={() => setRawOpen((v) => !v)}
            className="flex w-full items-center justify-between text-sm font-medium text-foreground"
          >
            <span>Raw markdown</span>
            <span className="text-xs text-foreground-secondary">{rawOpen ? "−" : "+"}</span>
          </button>
          {rawOpen && (
            <pre className="mt-3 max-h-96 overflow-auto rounded-md bg-surface-elevated p-3 text-xs text-foreground-secondary">
              {result.raw_markdown.length > 10000
                ? result.raw_markdown.slice(0, 10000) + "\n… truncated in UI"
                : result.raw_markdown}
            </pre>
          )}
        </div>
      )}

      <RawJsonView data={result} />
    </div>
  );
}

function CompanyResultPanel({ result }: { result: CompanyEnrichResult }) {
  const fields = result.fields;
  const hq = fields?.headquarters ?? undefined;
  const showConfidence = result.status === "ok" || result.status === "partial";
  const confidenceValue =
    showConfidence && result.confidence !== undefined && result.confidence !== null
      ? `${Math.round(result.confidence * 100)}%`
      : "—";
  const limitsApplied =
    typeof result.metadata?.limits_applied === "string"
      ? result.metadata.limits_applied
      : Array.isArray(result.metadata?.limits_applied)
        ? result.metadata?.limits_applied.join(", ")
        : undefined;

  return (
    <div className="space-y-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <ResultCard label="Source tool" value={result.source_tool || "—"} />
        <ResultCard label="Confidence" value={confidenceValue} />
        {result.metadata?.duration_ms !== undefined && (
          <ResultCard label="Duration" value={`${result.metadata.duration_ms} ms`} />
        )}
        <ResultCard label="Checked at" value={result.checked_at ? new Date(result.checked_at).toLocaleString() : "—"} />
      </div>

      {result.error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-3 text-sm text-danger">
          {result.error}
        </div>
      )}
      {result.reason && (
        <div className="rounded-md border border-warning/20 bg-warning/10 p-3 text-sm text-warning">
          {result.reason}
        </div>
      )}

      {fields && (
        <div className="space-y-4">
          <div className="rounded-md border border-border bg-surface p-4">
            <h4 className="mb-2 text-sm font-medium text-foreground">Identity</h4>
            <div className="grid gap-2 text-sm sm:grid-cols-2">
              {fields.name !== undefined && fields.name !== "" && (
                <ResultCard label="Name" value={fields.name} />
              )}
              {fields.legal_name && <ResultCard label="Legal name" value={fields.legal_name} />}
              {fields.domain && <ResultCard label="Domain" value={fields.domain} />}
              {fields.website && (
                <ResultCard
                  label="Website"
                  value={
                    <a
                      href={fields.website}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-primary hover:underline"
                    >
                      {fields.website}
                      <ExternalLink className="h-3 w-3" />
                    </a>
                  }
                />
              )}
            </div>
            {fields.name === undefined || fields.name === "" ? (
              <p className="mt-2 text-xs text-foreground-muted">
                Company name could not be derived from this input.
              </p>
            ) : null}
          </div>

          {fields.description && (
            <div className="rounded-md border border-border bg-surface p-4">
              <h4 className="mb-2 text-sm font-medium text-foreground">Description</h4>
              <p className="text-sm text-foreground">{fields.description}</p>
            </div>
          )}

          <div className="rounded-md border border-border bg-surface p-4">
            <h4 className="mb-2 text-sm font-medium text-foreground">Size &amp; maturity</h4>
            <div className="grid gap-2 text-sm sm:grid-cols-2 lg:grid-cols-4">
              {fields.employee_count !== undefined && fields.employee_count !== null && (
                <ResultCard label="Employee count" value={fields.employee_count} />
              )}
              {fields.employee_count_range && (
                <ResultCard label="Employee range" value={fields.employee_count_range} />
              )}
              {fields.founded !== undefined && fields.founded !== null && (
                <ResultCard label="Founded" value={fields.founded} />
              )}
            </div>
          </div>

          {hq && (hq.city || hq.state || hq.country || hq.address) && (
            <div className="rounded-md border border-border bg-surface p-4">
              <h4 className="mb-2 text-sm font-medium text-foreground">Headquarters</h4>
              <div className="grid gap-2 text-sm sm:grid-cols-2">
                {hq.city && <ResultCard label="City" value={hq.city} />}
                {hq.state && <ResultCard label="State" value={hq.state} />}
                {hq.country && <ResultCard label="Country" value={hq.country} />}
                {hq.address && <ResultCard label="Address" value={hq.address} />}
              </div>
            </div>
          )}

          {(fields.industry && fields.industry.length > 0) || (fields.tech_stack && fields.tech_stack.length > 0) ? (
            <div className="rounded-md border border-border bg-surface p-4">
              <h4 className="mb-2 text-sm font-medium text-foreground">Classification</h4>
              {fields.industry && fields.industry.length > 0 && (
                <div className="mb-2 flex flex-wrap gap-2">
                  {fields.industry.map((industry) => (
                    <Badge key={industry} variant="outline" className="text-xs">{industry}</Badge>
                  ))}
                </div>
              )}
              {fields.tech_stack && fields.tech_stack.length > 0 && (
                <div className="flex flex-wrap gap-2">
                  {fields.tech_stack.map((tech) => (
                    <Badge key={tech} variant="muted" className="text-xs">{tech}</Badge>
                  ))}
                </div>
              )}
            </div>
          ) : null}

          {fields.social_links && Object.keys(fields.social_links).length > 0 && (
            <div className="rounded-md border border-border bg-surface p-4">
              <h4 className="mb-2 text-sm font-medium text-foreground">Social links</h4>
              <div className="flex flex-wrap gap-2">
                {Object.entries(fields.social_links).map(([platform, url]) => (
                  <a
                    key={platform}
                    href={url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex items-center gap-1 rounded-md border border-border bg-surface-elevated px-2 py-1 text-xs text-foreground hover:bg-surface"
                  >
                    {platform}
                    <ExternalLink className="h-3 w-3" />
                  </a>
                ))}
              </div>
            </div>
          )}

          {fields.sources && fields.sources.length > 0 && (
            <div className="rounded-md border border-border bg-surface p-4">
              <h4 className="mb-2 text-sm font-medium text-foreground">Sources</h4>
              <RecordList label="" items={fields.sources} />
            </div>
          )}
        </div>
      )}

      {result.metadata && (
        <div className="rounded-md border border-border bg-surface p-4">
          <h4 className="mb-2 text-sm font-medium text-foreground">Metadata</h4>
          <div className="grid gap-2 text-sm sm:grid-cols-2">
            {result.metadata.backend && <ResultCard label="Backend" value={result.metadata.backend} />}
            {result.metadata.legal_basis && <ResultCard label="Legal basis" value={result.metadata.legal_basis} />}
            {result.metadata.permission_ref && <ResultCard label="Permission ref" value={result.metadata.permission_ref} />}
            {limitsApplied && <ResultCard label="Limits applied" value={limitsApplied} />}
          </div>
        </div>
      )}

      <RawJsonView data={result} />
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

function CRMReadinessSection({
  stage,
  readiness,
  loading,
  transitioning,
  onPromote,
  onDemote,
  onExport,
}: {
  stage: string;
  readiness?: ReadinessReport;
  loading: boolean;
  transitioning: boolean;
  onPromote: () => void;
  onDemote: () => void;
  onExport: () => void;
}) {
  const isCrmReady = stage === "crm_ready";

  return (
    <Card>
      <div className="flex items-start justify-between gap-4">
        <div>
          <h3 className="text-sm font-semibold text-foreground">CRM readiness</h3>
          <p className="text-xs text-foreground-secondary">
            {isCrmReady
              ? "This lead is CRM-ready and can be exported."
              : readiness
                ? readiness.ready
                  ? "All required checks pass. Promote to CRM-ready to export."
                  : "Some required checks have not passed yet."
                : "Checking readiness..."}
          </p>
          {readiness?.warning && (
            <div className="mt-2 flex items-start gap-2 text-xs text-warning">
              <AlertTriangle className="mt-0.5 h-3.5 w-3.5 shrink-0" />
              <span>{readiness.warning}</span>
            </div>
          )}
        </div>
        <div className="flex flex-col items-end gap-2 sm:flex-row sm:items-center">
          {isCrmReady ? (
            <>
              <Button
                size="sm"
                variant="secondary"
                onClick={onDemote}
                disabled={transitioning}
              >
                <TrendingDown className="mr-1.5 h-3.5 w-3.5" />
                Demote
              </Button>
              <Button
                size="sm"
                variant="primary"
                onClick={onExport}
                disabled={transitioning}
              >
                <Download className="mr-1.5 h-3.5 w-3.5" />
                Export stub
              </Button>
            </>
          ) : (
            <Button
              size="sm"
              variant="primary"
              onClick={onPromote}
              disabled={loading || transitioning || !readiness?.ready}
              title={readiness?.ready ? "Promote to CRM-ready" : "Lead does not meet CRM-ready requirements"}
            >
              <TrendingUp className="mr-1.5 h-3.5 w-3.5" />
              Promote to CRM-ready
            </Button>
          )}
        </div>
      </div>

      {loading && (
        <div className="mt-4 space-y-2">
          <Skeleton className="h-4 w-3/4" />
          <Skeleton className="h-4 w-1/2" />
        </div>
      )}

      {readiness && (
        <div className="mt-4 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {readiness.checks.map((check) => (
            <ReadinessCheckRow key={check.name} check={check} />
          ))}
        </div>
      )}
    </Card>
  );
}

function ReadinessCheckRow({ check }: { check: ReadinessCheck }) {
  return (
    <div className={cn(
      "flex items-start gap-2 rounded-md border p-2 text-xs",
      check.pass
        ? "border-success/20 bg-success/10 text-success"
        : check.required
          ? "border-danger/20 bg-danger/10 text-danger"
          : "border-warning/20 bg-warning/10 text-warning"
    )}>
      <span className="mt-0.5 shrink-0">
        {check.pass ? "✓" : check.required ? "✗" : "!"}
      </span>
      <div>
        <span className="font-medium capitalize">{check.name.replace(/_/g, " ")}</span>
        <p className="text-foreground-secondary">{check.message}</p>
      </div>
    </div>
  );
}
