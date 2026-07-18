"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { Play, Search, X } from "lucide-react";
import { useRuns } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/Table";
import type { PipelineRun } from "@/lib/api/types";

const statusOptions = [
  { value: "", label: "All statuses" },
  { value: "running", label: "Running" },
  { value: "completed", label: "Completed" },
  { value: "failed", label: "Failed" },
  { value: "partial", label: "Partial" },
];

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

function formatDate(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

export default function RunsPage() {
  const router = useRouter();
  const { data, isLoading, error } = useRuns({ page_size: 50 });
  const [statusFilter, setStatusFilter] = useState("");
  const [query, setQuery] = useState("");

  const filtered = useMemo(() => {
    if (!data?.data) return [];
    return data.data.filter((run: PipelineRun) => {
      if (statusFilter && run.status !== statusFilter) return false;
      if (query) {
        const q = query.toLowerCase();
        return (
          run.id.toLowerCase().includes(q) ||
          run.modules_executed.some((m) => m.toLowerCase().includes(q)) ||
          (run.error && run.error.toLowerCase().includes(q))
        );
      }
      return true;
    });
  }, [data, statusFilter, query]);

  const hasFilters = query !== "" || statusFilter !== "";

  return (
    <div className="space-y-6">
      <PageHeader title="Runs" description="Pipeline run history and execution records." />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground-muted" />
          <input
            type="text"
            placeholder="Search by ID, module, or error..."
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            className="w-full rounded-md border border-border bg-surface py-2 pl-9 pr-8 text-sm text-foreground placeholder:text-foreground-muted focus:border-primary focus:outline-none focus:ring-1 focus:ring-primary/50"
          />
          {query && (
            <button
              onClick={() => setQuery("")}
              className="absolute right-2 top-1/2 -translate-y-1/2 text-foreground-muted hover:text-foreground"
              aria-label="Clear search"
            >
              <X className="h-4 w-4" />
            </button>
          )}
        </div>
        <Select
          options={statusOptions}
          value={statusFilter}
          onChange={(e) => setStatusFilter(e.target.value)}
          className="w-full sm:w-44"
        />
      </div>

      {error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger" role="alert">
          Failed to load runs: {error.message}
        </div>
      )}

      <Card>
        {isLoading ? (
          <div className="space-y-3 p-4">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : data?.data.length === 0 ? (
          <EmptyState
            icon={Play}
            title="No runs yet"
            description="Pipeline runs will appear here after you run modules on leads."
          />
        ) : filtered.length === 0 && hasFilters ? (
          <EmptyState
            icon={Search}
            title="No matching runs"
            description="Try a different search term or status filter."
          />
        ) : (
          <>
            {/* Desktop table */}
            <div className="hidden sm:block">
              <Table>
                <TableHead>
                  <TableRow>
                    <TableHeader>Run ID</TableHeader>
                    <TableHeader>Status</TableHeader>
                    <TableHeader>Type</TableHeader>
                    <TableHeader>Started</TableHeader>
                    <TableHeader>Finished</TableHeader>
                    <TableHeader>Modules</TableHeader>
                    <TableHeader>Leads</TableHeader>
                    <TableHeader>Error</TableHeader>
                  </TableRow>
                </TableHead>
                <TableBody>
                  {filtered.map((run) => (
                    <TableRow
                      key={run.id}
                      onClick={() => router.push(`/runs/${run.id}`)}
                      className="cursor-pointer"
                    >
                      <TableCell className="font-mono text-xs">
                        {run.id.slice(0, 8)}
                      </TableCell>
                      <TableCell>
                        <Badge variant={variantForStatus(run.status)}>{run.status}</Badge>
                      </TableCell>
                      <TableCell className="capitalize">{run.type}</TableCell>
                      <TableCell className="text-xs">{formatDate(run.started_at)}</TableCell>
                      <TableCell className="text-xs">{formatDate(run.finished_at)}</TableCell>
                      <TableCell className="text-xs">
                        {run.modules_executed.join(", ") || "—"}
                      </TableCell>
                      <TableCell>{run.lead_ids.length}</TableCell>
                      <TableCell className="max-w-[12rem] truncate text-xs text-danger">
                        {run.error || "—"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </div>

            {/* Mobile card list */}
            <div className="space-y-3 p-3 sm:hidden">
              {filtered.map((run) => (
                <button
                  key={run.id}
                  onClick={() => router.push(`/runs/${run.id}`)}
                  className="w-full rounded-md border border-border bg-surface p-3 text-left transition-colors hover:border-primary/50"
                >
                  <div className="flex items-center justify-between">
                    <span className="font-mono text-xs text-foreground">
                      {run.id.slice(0, 8)}
                    </span>
                    <Badge variant={variantForStatus(run.status)}>{run.status}</Badge>
                  </div>
                  <div className="mt-2 text-xs text-foreground-muted">
                    <span className="capitalize">{run.type}</span>
                    {" — "}
                    {formatDate(run.started_at)}
                  </div>
                  <div className="mt-1 text-xs text-foreground-muted">
                    {run.modules_executed.join(", ") || "No modules"}
                    {" — "}
                    {run.lead_ids.length} lead{run.lead_ids.length !== 1 ? "s" : ""}
                  </div>
                  {run.error && (
                    <div className="mt-1 truncate text-xs text-danger">{run.error}</div>
                  )}
                </button>
              ))}
            </div>
          </>
        )}
      </Card>
    </div>
  );
}
