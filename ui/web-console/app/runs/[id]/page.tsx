"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, AlertCircle, Clock } from "lucide-react";
import { useRun } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";

function variantForStatus(status: string) {
  switch (status) {
    case "completed":
      return "success" as const;
    case "running":
      return "primary" as const;
    case "partial":
      return "warning" as const;
    case "failed":
      return "danger" as const;
    default:
      return "outline" as const;
  }
}

function formatTimestamp(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString();
}

export default function RunDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: run, isLoading, error } = useRun(id);

  // Gentle polling while run is in non-terminal state
  const isTerminal = run?.status === "completed" || run?.status === "failed" || run?.status === "partial";

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-48 w-full" />
      </div>
    );
  }

  if (error || !run) {
    return (
      <div className="space-y-6">
        <PageHeader title="Run not found" />
        <Card className="p-6">
          <EmptyState
            icon={AlertCircle}
            title="Failed to load run"
            description={error?.message || "This run does not exist or the API is unreachable."}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader
        title={`Run ${run.id.slice(0, 8)}`}
        description={run.error || "Pipeline execution record"}
      >
        <Link
          href="/runs"
          className="inline-flex items-center rounded-md bg-transparent px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-surface-elevated focus:outline-none focus:ring-2 focus:ring-primary/50"
        >
          <ArrowLeft className="mr-1.5 h-4 w-4" />
          Back to runs
        </Link>
      </PageHeader>

      {!isTerminal && (
        <div className="flex items-center gap-2 rounded-md border border-primary/20 bg-primary/5 p-3 text-sm text-foreground-secondary">
          <Clock className="h-4 w-4 shrink-0 text-primary" />
          <span>This run is still in progress. Status updates automatically.</span>
        </div>
      )}

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2 space-y-4">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={variantForStatus(run.status)}>{run.status}</Badge>
            <Badge variant="secondary" className="capitalize">
              {run.type}
            </Badge>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Run ID" value={<span className="font-mono text-xs">{run.id}</span>} />
            <Field label="Started" value={formatTimestamp(run.started_at)} />
            <Field
              label="Finished"
              value={formatTimestamp(run.finished_at)}
            />
            <Field label="Legal basis" value={run.legal_basis || "—"} />
            <Field label="Permission refs" value={run.permission_refs.join(", ") || "—"} />
            <Field label="Audit event count" value={run.audit_event_ids.length} />
          </div>

          {run.error && (
            <div className="rounded-md border border-danger/20 bg-danger/10 p-3 text-sm text-danger">
              <span className="font-medium">Error:</span> {run.error}
            </div>
          )}
        </Card>

        <div className="space-y-6">
          <Card>
            <h3 className="mb-3 text-sm font-semibold text-foreground">Modules executed</h3>
            {run.modules_executed.length === 0 ? (
              <p className="text-sm text-foreground-muted">No modules recorded for this run.</p>
            ) : (
              <ul className="space-y-1.5">
                {run.modules_executed.map((mod) => (
                  <li key={mod} className="flex items-center gap-2 text-sm">
                    <span className="h-1.5 w-1.5 rounded-full bg-primary" />
                    <Link
                      href={`/modules/${mod}`}
                      className="text-primary hover:underline"
                    >
                      {mod}
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Card>

          <Card>
            <h3 className="mb-3 text-sm font-semibold text-foreground">Related leads</h3>
            {run.lead_ids.length === 0 ? (
              <p className="text-sm text-foreground-muted">No leads in this run.</p>
            ) : (
              <ul className="space-y-1.5">
                {run.lead_ids.map((leadId) => (
                  <li key={leadId}>
                    <Link
                      href={`/leads/${leadId}`}
                      className="font-mono text-xs text-primary hover:underline"
                    >
                      {leadId.slice(0, 12)}...
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      </div>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold text-foreground">Audit events</h3>
        {run.audit_event_ids.length === 0 ? (
          <p className="text-sm text-foreground-muted">
            No per-run audit events in payload. Check individual lead detail pages for audit trails.
          </p>
        ) : (
          <div className="space-y-1.5">
            <p className="text-xs text-foreground-muted">
              {run.audit_event_ids.length} audit event{run.audit_event_ids.length !== 1 ? "s" : ""} recorded.
              View full audit details on individual lead pages.
            </p>
            <ul className="grid gap-1 sm:grid-cols-2 lg:grid-cols-3">
              {run.audit_event_ids.slice(0, 12).map((eventId) => (
                <li key={eventId} className="font-mono text-xs text-foreground-muted">
                  {eventId.slice(0, 12)}
                </li>
              ))}
              {run.audit_event_ids.length > 12 && (
                <li className="text-xs text-foreground-muted">
                  +{run.audit_event_ids.length - 12} more
                </li>
              )}
            </ul>
          </div>
        )}
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
