"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, AlertCircle } from "lucide-react";
import { useRun } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";

export default function RunDetailPage() {
  const { id } = useParams<{ id: string }>();
  const { data: run, isLoading, error } = useRun(id);

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
            description={error?.message || "This run does not exist."}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader title={`Run ${run.id.slice(0, 8)}`} description={run.error || "Pipeline execution record"}>
        <Link href="/runs">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-1.5 h-4 w-4" />
            Back to runs
          </Button>
        </Link>
      </PageHeader>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2 space-y-4">
          <div className="flex items-center gap-2">
            <Badge variant={variantForStatus(run.status)}>{run.status}</Badge>
            <Badge variant="secondary" className="capitalize">
              {run.type}
            </Badge>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Started" value={new Date(run.started_at).toLocaleString()} />
            <Field
              label="Finished"
              value={run.finished_at ? new Date(run.finished_at).toLocaleString() : "—"}
            />
            <Field label="Legal basis" value={run.legal_basis} />
            <Field label="Permission refs" value={run.permission_refs.join(", ") || "—"} />
            <Field label="Modules executed" value={run.modules_executed.join(", ") || "—"} />
            <Field label="Audit events" value={run.audit_event_ids.length} />
          </div>
        </Card>

        <Card>
          <h3 className="mb-3 text-sm font-semibold text-foreground">Leads</h3>
          {run.lead_ids.length === 0 ? (
            <p className="text-sm text-foreground-muted">No leads in this run.</p>
          ) : (
            <ul className="space-y-2">
              {run.lead_ids.map((leadId) => (
                <li key={leadId}>
                  <Link href={`/leads/${leadId}`} className="text-sm text-primary hover:underline">
                    {leadId}
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </div>
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

function variantForStatus(status: string) {
  switch (status) {
    case "completed":
      return "success";
    case "running":
      return "primary";
    case "partial":
      return "warning";
    case "failed":
      return "danger";
    default:
      return "outline";
  }
}
