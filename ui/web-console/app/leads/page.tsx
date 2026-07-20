"use client";

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  Plus,
  Search,
  Users,
  X,
  Loader2,
  AlertTriangle,
} from "lucide-react";
import { useLeads, useModules, useRunPipeline, useRunStatus } from "@/lib/api/hooks";
import { ApiClientError } from "@/lib/api/client";
import { useQueryClient } from "@tanstack/react-query";
import { useToast } from "@/components/providers/ToastProvider";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyWorkspaceState } from "@/components/ui/EmptyWorkspaceState";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/Table";
import { LeadReadinessCell } from "@/components/leads/LeadReadinessCell";
import { PermissionReferenceInline } from "@/components/leads/PermissionReferenceField";
import type { LeadSummary, RiskLevel, ModuleName } from "@/lib/api/types";

const stageOptions = [
  { value: "", label: "All stages" },
  { value: "raw", label: "Raw" },
  { value: "enriched", label: "Enriched" },
  { value: "validated", label: "Validated" },
  { value: "crm_ready", label: "CRM ready" },
];

const riskOptions = [
  { value: "", label: "All risk" },
  { value: "low", label: "Low" },
  { value: "medium", label: "Medium" },
  { value: "high", label: "High" },
  { value: "unknown", label: "Unknown" },
];

const moduleStatusOptions = [
  { value: "", label: "All module status" },
  { value: "ok", label: "ok" },
  { value: "partial", label: "partial" },
  { value: "unknown", label: "unknown" },
  { value: "skipped", label: "skipped" },
  { value: "error", label: "error" },
  { value: "not_run", label: "not run" },
];

function riskVariant(level: RiskLevel) {
  switch (level) {
    case "low":
      return "success";
    case "medium":
      return "warning";
    case "high":
      return "danger";
    default:
      return "muted";
  }
}

function formatDate(iso?: string) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

function useAvailableModules() {
  const { data: modules } = useModules();
  return useMemo(() => {
    return modules?.filter((m) => m.dev_status === "available") ?? [];
  }, [modules]);
}

export default function LeadsPage() {
  const router = useRouter();
  const { addToast } = useToast();
  const queryClient = useQueryClient();
  const [filters, setFilters] = useState({
    stage: "",
    risk: "",
    module_status: "",
    q: "",
    page: 1,
    page_size: 25,
  });
  const [selected, setSelected] = useState<Set<string>>(new Set());
  const [activeBulkRunId, setActiveBulkRunId] = useState<string | null>(null);

  const { data, isLoading, error } = useLeads(filters);
  const availableModules = useAvailableModules();
  const pipeline = useRunPipeline();
  const { data: bulkRunStatus } = useRunStatus(activeBulkRunId ?? undefined);

  const leads: LeadSummary[] = data?.data ?? [];
  const meta = data?.meta;
  const hasActiveFilters =
    filters.stage || filters.risk || filters.module_status || filters.q;

  const allSelected = leads.length > 0 && selected.size === leads.length;

  const selectedLeads = leads.filter((l) => selected.has(l.id));

  const canRunBulk =
    selected.size > 0 && selectedLeads.every((l) => l.permission_ref);
  const bulkRunActive =
    !!activeBulkRunId &&
    (!bulkRunStatus || bulkRunStatus.status === "queued" || bulkRunStatus.status === "running");

  useEffect(() => {
    if (!activeBulkRunId || !bulkRunStatus) return;
    if (bulkRunStatus.status !== "completed" && bulkRunStatus.status !== "partial" && bulkRunStatus.status !== "failed") return;

    setActiveBulkRunId(null);
    queryClient.invalidateQueries({ queryKey: ["leads"] });
    addToast(`Bulk run ${bulkRunStatus.status}`, bulkRunStatus.status === "completed" ? "success" : "warning");
  }, [activeBulkRunId, bulkRunStatus, queryClient, addToast]);

  const updateFilter = (key: string, value: string) => {
    setFilters((prev) => ({ ...prev, [key]: value, page: 1 }));
    setSelected(new Set());
  };

  const clearFilters = () => {
    setFilters({
      stage: "",
      risk: "",
      module_status: "",
      q: "",
      page: 1,
      page_size: 25,
    });
    setSelected(new Set());
  };

  const toggleSelect = (id: string) => {
    const next = new Set(selected);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setSelected(next);
  };

  const toggleAll = () => {
    if (allSelected) {
      setSelected(new Set());
    } else {
      setSelected(new Set(leads.map((l) => l.id)));
    }
  };

  const clearSelection = () => setSelected(new Set());

  const runBulk = async (moduleName: ModuleName, displayName: string) => {
    if (!canRunBulk) return;
    try {
      const res = await pipeline.mutateAsync({
        lead_ids: Array.from(selected),
        modules: [moduleName],
      });
      setActiveBulkRunId(res.run_id);
      addToast(
        <span>
          Started {displayName} for {selected.size} lead(s).{" "}
          <Link
            href={`/runs/${res.run_id}`}
            className="underline hover:no-underline"
          >
            View run {res.run_id.slice(0, 8)}…
          </Link>
        </span>,
        "success"
      );
    } catch (err) {
      if (err instanceof ApiClientError && err.status === 409) {
        addToast("A selected lead already has an active run. Wait for it to finish before starting a bulk run.", "warning");
        return;
      }
      addToast(
        err instanceof Error ? err.message : "Pipeline run failed",
        "danger"
      );
    }
  };

  const handleRowClick = (id: string) => {
    router.push(`/leads/${id}`);
  };

  const handleRowKeyDown = (e: React.KeyboardEvent, id: string) => {
    if (e.key === "Enter" || e.key === " ") {
      e.preventDefault();
      handleRowClick(id);
    }
  };

  const emptyState = (
    <EmptyWorkspaceState
      icon={Users}
      title="No leads yet"
      description="Create a permissioned lead before running checks, or explore available modules to see what the platform can do."
      primaryAction={{ label: "Create lead", href: "/leads/create" }}
      secondaryAction={{ label: "Explore modules", href: "/modules" }}
    />
  );

  return (
    <div className="space-y-6">
      <PageHeader
        title="Leads"
        description="Manage and inspect permissioned leads."
      >
        <Link
          href="/leads/create"
          className="inline-flex items-center justify-center rounded-md bg-primary px-4 py-2 text-sm font-medium text-background transition-colors hover:brightness-110 focus:outline-none focus:ring-2 focus:ring-primary/50"
        >
          <Plus className="mr-1.5 h-4 w-4" />
          Create lead
        </Link>
      </PageHeader>

      {activeBulkRunId && (
        <Card className="flex flex-wrap items-center justify-between gap-3 border-primary/20 bg-primary/5">
          <div className="text-sm text-foreground">
            <span className="font-medium">Active bulk run</span>{" "}
            <span className="font-mono text-xs">{activeBulkRunId.slice(0, 8)}</span>{" "}
            <span className="capitalize text-foreground-secondary">
              {bulkRunStatus?.status || "queued"}
            </span>
          </div>
          <Link
            href={`/runs/${activeBulkRunId}`}
            className="text-sm font-medium text-primary hover:underline"
          >
            View run
          </Link>
        </Card>
      )}

      {isLoading ? (
        <Card className="p-4">
          <div className="space-y-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        </Card>
      ) : error ? (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger">
          Failed to load leads: {error.message}
        </div>
      ) : leads.length === 0 ? (
        <Card className="p-4">
          {hasActiveFilters ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <p className="text-foreground">No leads match your filters.</p>
              <Button
                variant="ghost"
                size="sm"
                onClick={clearFilters}
                className="mt-3"
              >
                <X className="mr-1 h-4 w-4" />
                Clear filters
              </Button>
            </div>
          ) : (
            emptyState
          )}
        </Card>
      ) : (
        <>
          <Card className="p-4">
            <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-5">
              <div className="sm:col-span-2 lg:col-span-2">
                <div className="relative">
                  <Input
                    label="Search"
                    placeholder="Name, email, company, domain, URL"
                    value={filters.q}
                    onChange={(e) => updateFilter("q", e.target.value)}
                  />
                  <Search className="pointer-events-none absolute right-3 top-8 h-4 w-4 text-foreground-muted" />
                </div>
              </div>
              <Select
                label="Stage"
                options={stageOptions}
                value={filters.stage}
                onChange={(e) => updateFilter("stage", e.target.value)}
              />
              <Select
                label="Risk level"
                options={riskOptions}
                value={filters.risk}
                onChange={(e) => updateFilter("risk", e.target.value)}
              />
              <Select
                label="Module status"
                options={moduleStatusOptions}
                value={filters.module_status}
                onChange={(e) => updateFilter("module_status", e.target.value)}
              />
            </div>
          </Card>

          {selected.size > 0 && (
            <Card className="flex flex-wrap items-center justify-between gap-3 p-3">
              <div className="flex items-center gap-2 text-sm text-foreground">
                <span className="font-medium">{selected.size}</span> selected
                {!canRunBulk && (
                  <span
                    className="inline-flex items-center gap-1 text-xs text-warning"
                    title="All selected leads must have a permission reference"
                  >
                    <AlertTriangle className="h-3.5 w-3.5" />
                    Missing permission ref
                  </span>
                )}
                <Button
                  size="sm"
                  variant="ghost"
                  onClick={clearSelection}
                >
                  <X className="mr-1 h-3.5 w-3.5" />
                  Clear
                </Button>
              </div>
              <div className="flex flex-wrap gap-2">
                {availableModules.map((m) => (
                  <Button
                    key={m.name}
                    size="sm"
                    variant="secondary"
                    disabled={!canRunBulk || pipeline.isPending || bulkRunActive}
                    onClick={() => runBulk(m.name, m.display_name)}
                    title={
                      canRunBulk
                        ? `Run ${m.display_name} on selected leads`
                        : "All selected leads must have a permission reference"
                    }
                  >
                    {pipeline.isPending ? (
                      <Loader2 className="mr-1 h-3.5 w-3.5 animate-spin" />
                    ) : null}
                    Run {m.display_name}
                  </Button>
                ))}
              </div>
            </Card>
          )}

          {/* Desktop table */}
          <Card className="hidden md:block">
            <Table>
              <TableHead>
                <TableRow>
                  <TableHeader className="w-10">
                    <input
                      type="checkbox"
                      aria-label="Select all leads"
                      checked={allSelected}
                      onChange={toggleAll}
                      className="h-4 w-4 rounded border-border bg-surface text-primary focus:ring-primary/50"
                    />
                  </TableHeader>
                  <TableHeader>Lead</TableHeader>
                  <TableHeader>Stage</TableHeader>
                  <TableHeader>Risk</TableHeader>
                  <TableHeader>Permission ref</TableHeader>
                  <TableHeader>Readiness</TableHeader>
                  <TableHeader>Updated</TableHeader>
                </TableRow>
              </TableHead>
              <TableBody>
                {leads.map((lead) => (
                  <TableRow
                    key={lead.id}
                    onClick={() => handleRowClick(lead.id)}
                    onKeyDown={(e) => handleRowKeyDown(e, lead.id)}
                    tabIndex={0}
                    className="cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary/50"
                  >
                    <TableCell onClick={(e) => e.stopPropagation()}>
                      <input
                        type="checkbox"
                        aria-label={`Select lead ${lead.name || lead.id}`}
                        checked={selected.has(lead.id)}
                        onChange={() => toggleSelect(lead.id)}
                        className="h-4 w-4 rounded border-border bg-surface text-primary focus:ring-primary/50"
                      />
                    </TableCell>
                    <TableCell>
                      <div className="font-medium text-foreground">
                        {lead.name || "—"}
                      </div>
                      <div className="text-xs text-foreground-muted">
                        {lead.email || lead.url || lead.domain || lead.company || lead.id}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="capitalize">
                        {lead.stage.replace(/_/g, " ")}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <Badge variant={riskVariant(lead.risk_level)}>
                        {lead.risk_level}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <PermissionReferenceInline
                        permissionRef={lead.permission_ref}
                      />
                    </TableCell>
                    <TableCell>
                      <LeadReadinessCell lead={lead} />
                    </TableCell>
                    <TableCell>
                      <span className="text-sm text-foreground-secondary">
                        {formatDate(lead.updated_at)}
                      </span>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </Card>

          {/* Mobile card list */}
          <div className="space-y-3 md:hidden">
            {leads.map((lead) => (
              <Card
                key={lead.id}
                className="p-4 focus:outline-none focus:ring-2 focus:ring-primary/50"
                tabIndex={0}
                onClick={() => handleRowClick(lead.id)}
                onKeyDown={(e) => handleRowKeyDown(e, lead.id)}
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0 flex-1">
                    <div className="font-medium text-foreground">
                      {lead.name || "—"}
                    </div>
                    <div className="text-xs text-foreground-muted">
                      {lead.email || lead.url || lead.domain || lead.company || lead.id}
                    </div>
                  </div>
                  <input
                    type="checkbox"
                    aria-label={`Select lead ${lead.name || lead.id}`}
                    checked={selected.has(lead.id)}
                    onChange={(e) => {
                      e.stopPropagation();
                      toggleSelect(lead.id);
                    }}
                    className="h-4 w-4 rounded border-border bg-surface text-primary focus:ring-primary/50"
                  />
                </div>
                <div className="mt-3 grid grid-cols-2 gap-2 text-sm">
                  <div>
                    <span className="text-foreground-muted">Stage</span>
                    <div>
                      <Badge variant="secondary" className="capitalize">
                        {lead.stage.replace(/_/g, " ")}
                      </Badge>
                    </div>
                  </div>
                  <div>
                    <span className="text-foreground-muted">Risk</span>
                    <div>
                      <Badge variant={riskVariant(lead.risk_level)}>
                        {lead.risk_level}
                      </Badge>
                    </div>
                  </div>
                </div>
                <div className="mt-3">
                  <span className="text-foreground-muted">Permission ref</span>
                  <div>
                    <PermissionReferenceInline permissionRef={lead.permission_ref} />
                  </div>
                </div>
                <div className="mt-3">
                  <span className="text-foreground-muted">Readiness</span>
                  <LeadReadinessCell lead={lead} />
                </div>
                <div className="mt-3 text-xs text-foreground-muted">
                  Updated {formatDate(lead.updated_at)}
                </div>
              </Card>
            ))}
          </div>

          {meta && meta.total > 0 && (
            <div className="flex items-center justify-between text-sm text-foreground-muted">
              <span>
                Showing {(meta.page - 1) * meta.page_size + 1} -{" "}
                {Math.min(meta.page * meta.page_size, meta.total)} of{" "}
                {meta.total}
              </span>
              <div className="flex gap-2">
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={meta.page <= 1}
                  onClick={() =>
                    setFilters((prev) => ({ ...prev, page: meta.page - 1 }))
                  }
                >
                  Previous
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  disabled={meta.page * meta.page_size >= meta.total}
                  onClick={() =>
                    setFilters((prev) => ({ ...prev, page: meta.page + 1 }))
                  }
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </>
      )}
    </div>
  );
}
