"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import {
  ScrollText,
  Search,
  X,
  ChevronDown,
  ChevronRight,
  ChevronLeft,
} from "lucide-react";
import { useAudit } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import type { AuditEvent } from "@/lib/api/types";

const moduleOptions = [
  { value: "", label: "All modules" },
  { value: "email-validate", label: "email-validate" },
  { value: "phone-validate", label: "phone-validate" },
  { value: "domain-intel", label: "domain-intel" },
  { value: "social-footprint", label: "social-footprint" },
  { value: "extraction", label: "extraction" },
  { value: "pipeline", label: "pipeline" },
];

const statusOptions = [
  { value: "", label: "All statuses" },
  { value: "ok", label: "ok" },
  { value: "unknown", label: "unknown" },
  { value: "skipped", label: "skipped" },
];

const PAGE_SIZE = 25;

function statusVariant(status: string) {
  switch (status) {
    case "ok":
      return "success" as const;
    case "skipped":
      return "warning" as const;
    case "unknown":
      return "muted" as const;
    default:
      return "outline" as const;
  }
}

function formatTimestamp(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  return d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
}

function formatStderr(raw: string): string {
  try {
    const parsed = JSON.parse(raw);
    return JSON.stringify(parsed, null, 2);
  } catch {
    return raw;
  }
}

function subjectSummary(event: AuditEvent): string {
  if (!event.subject) return "—";
  const parts: string[] = [];
  if (event.subject.email) parts.push(event.subject.email);
  if (event.subject.domain) parts.push(event.subject.domain);
  if (event.subject.url) parts.push(event.subject.url);
  if (event.subject.phone_redacted) parts.push(event.subject.phone_redacted);
  if (event.subject.handle) parts.push(`@${event.subject.handle}`);
  return parts.join(", ") || "—";
}

export default function AuditPage() {
  const [moduleFilter, setModuleFilter] = useState("");
  const [statusFilter, setStatusFilter] = useState("");
  const [page, setPage] = useState(1);
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const apiParams = useMemo(() => {
    const p: Record<string, string | number> = { page, page_size: PAGE_SIZE };
    if (moduleFilter) p.module = moduleFilter;
    if (statusFilter) p.status = statusFilter;
    return p;
  }, [moduleFilter, statusFilter, page]);

  const { data, isLoading, error } = useAudit(apiParams);

  const filtered = useMemo(() => {
    if (!data?.data) return [];
    if (!query) return data.data;
    const q = query.toLowerCase();
    return data.data.filter(
      (e) =>
        e.id.toLowerCase().includes(q) ||
        e.lead_id.toLowerCase().includes(q) ||
        e.module.toLowerCase().includes(q) ||
        e.tool.toLowerCase().includes(q) ||
        (e.subject?.email && e.subject.email.toLowerCase().includes(q)) ||
        (e.subject?.domain && e.subject.domain.toLowerCase().includes(q)) ||
        (e.subject?.url && e.subject.url.toLowerCase().includes(q))
    );
  }, [data, query]);

  const totalPages = data?.meta ? Math.ceil(data.meta.total / PAGE_SIZE) : 1;

  const toggle = (id: string) => {
    setExpanded((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  const handleModuleChange = (v: string) => {
    setModuleFilter(v);
    setPage(1);
  };
  const handleStatusChange = (v: string) => {
    setStatusFilter(v);
    setPage(1);
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Audit log"
        description="Immutable evidence of module runs. Every event records GDPR Art.6(1)(f) legal basis."
      />

      {/* Filters */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground-muted" />
          <input
            type="text"
            placeholder="Search by ID, lead, module, or subject..."
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
          options={moduleOptions}
          value={moduleFilter}
          onChange={(e) => handleModuleChange(e.target.value)}
          className="w-full sm:w-44"
        />
        <Select
          options={statusOptions}
          value={statusFilter}
          onChange={(e) => handleStatusChange(e.target.value)}
          className="w-full sm:w-36"
        />
      </div>

      {error && (
        <div
          className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger"
          role="alert"
        >
          Failed to load audit events: {error.message}
        </div>
      )}

      <Card>
        {isLoading ? (
          <div className="space-y-3 p-4">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-8 w-full" />
          </div>
        ) : data?.data.length === 0 ? (
          <EmptyState
            icon={ScrollText}
            title="No audit events yet"
            description="Audit events will appear here after running modules on leads."
          />
        ) : filtered.length === 0 && query ? (
          <EmptyState
            icon={Search}
            title="No matching events"
            description="Try a different search term."
          />
        ) : (
          <>
            {query && (
              <div className="px-4 py-2 text-xs text-foreground-muted">
                Filtering this page only ({filtered.length} event{filtered.length !== 1 ? "s" : ""}).
                Clear search to use server pagination.
              </div>
            )}
            {/* Desktop list */}
            <div className="hidden sm:block">
              <div className="divide-y divide-border">
                {/* Header */}
                <div className="grid grid-cols-[1fr_7rem_7rem_6rem_5rem_10rem_2rem] gap-2 px-4 py-2 text-xs font-medium text-foreground-muted">
                  <span>Timestamp</span>
                  <span>Module</span>
                  <span>Tool</span>
                  <span>Status</span>
                  <span>Basis</span>
                  <span>Subject</span>
                  <span />
                </div>
                {filtered.map((event) => (
                  <div key={event.id}>
                    <button
                      onClick={() => toggle(event.id)}
                      className="grid w-full grid-cols-[1fr_7rem_7rem_6rem_5rem_10rem_2rem] items-center gap-2 px-4 py-3 text-left text-sm transition-colors hover:bg-surface-elevated"
                    >
                      <span className="text-xs text-foreground">
                        {formatTimestamp(event.checked_at)}
                      </span>
                      <span className="truncate font-medium text-foreground">
                        {event.module}
                      </span>
                      <span className="truncate text-xs text-foreground-muted">
                        {event.tool || "—"}
                      </span>
                      <span>
                        <Badge variant={statusVariant(event.status)}>
                          {event.status}
                        </Badge>
                      </span>
                      <span className="truncate text-xs text-foreground-muted">
                        {event.legal_basis}
                      </span>
                      <span className="truncate text-xs text-foreground-muted">
                        {subjectSummary(event)}
                      </span>
                      <span>
                        {expanded.has(event.id) ? (
                          <ChevronDown className="h-4 w-4 text-foreground-muted" />
                        ) : (
                          <ChevronRight className="h-4 w-4 text-foreground-muted" />
                        )}
                      </span>
                    </button>
                    {expanded.has(event.id) && (
                      <div className="border-t border-border bg-surface-elevated px-4 py-3 text-xs">
                        <div className="grid gap-2 sm:grid-cols-2 lg:grid-cols-4">
                          <Field label="Event ID" value={event.id} mono />
                          <Field
                            label="Lead"
                            value={
                              <Link
                                href={`/leads/${event.lead_id}`}
                                className="text-primary hover:underline"
                              >
                                {event.lead_id.slice(0, 12)}...
                              </Link>
                            }
                          />
                          {event.run_id && (
                            <Field
                              label="Run"
                              value={
                                <Link
                                  href={`/runs/${event.run_id}`}
                                  className="text-primary hover:underline"
                                >
                                  {event.run_id.slice(0, 12)}...
                                </Link>
                              }
                            />
                          )}
                          <Field label="Legal basis" value={event.legal_basis} />
                        </div>
                        {event.raw_stderr_json && (
                          <div className="mt-3">
                            <div className="mb-1 text-foreground-muted">
                              raw_stderr_json
                            </div>
                            <pre className="overflow-x-auto rounded bg-surface p-2 font-mono text-xs text-foreground-muted">
                              {formatStderr(event.raw_stderr_json)}
                            </pre>
                          </div>
                        )}
                        {!event.raw_stderr_json && (
                          <p className="mt-2 text-foreground-muted">
                            No raw stderr payload for this event.
                          </p>
                        )}
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>

            {/* Mobile card list */}
            <div className="space-y-2 p-3 sm:hidden">
              {filtered.map((event) => (
                <div
                  key={event.id}
                  className="rounded-md border border-border bg-surface"
                >
                  <button
                    onClick={() => toggle(event.id)}
                    className="w-full p-3 text-left"
                  >
                    <div className="flex items-center justify-between">
                      <span className="text-xs font-medium text-foreground">
                        {event.module}
                      </span>
                      <Badge variant={statusVariant(event.status)}>
                        {event.status}
                      </Badge>
                    </div>
                    <div className="mt-1 text-xs text-foreground-muted">
                      {formatTimestamp(event.checked_at)}
                      {event.tool && ` — ${event.tool}`}
                    </div>
                    <div className="mt-1 text-xs text-foreground-muted">
                      {event.legal_basis} — {subjectSummary(event)}
                    </div>
                  </button>
                  {expanded.has(event.id) && (
                    <div className="border-t border-border bg-surface-elevated p-3 text-xs">
                      <div className="space-y-1">
                        <div>
                          <span className="text-foreground-muted">Lead: </span>
                          <Link
                            href={`/leads/${event.lead_id}`}
                            className="text-primary hover:underline"
                          >
                            {event.lead_id.slice(0, 12)}...
                          </Link>
                        </div>
                        {event.run_id && (
                          <div>
                            <span className="text-foreground-muted">Run: </span>
                            <Link
                              href={`/runs/${event.run_id}`}
                              className="text-primary hover:underline"
                            >
                              {event.run_id.slice(0, 12)}...
                            </Link>
                          </div>
                        )}
                      </div>
                      {event.raw_stderr_json && (
                        <pre className="mt-2 overflow-x-auto rounded bg-surface p-2 font-mono text-xs text-foreground-muted">
                          {formatStderr(event.raw_stderr_json)}
                        </pre>
                      )}
                      {!event.raw_stderr_json && (
                        <p className="mt-2 text-foreground-muted">
                          No raw stderr payload.
                        </p>
                      )}
                    </div>
                  )}
                </div>
              ))}
            </div>
          </>
        )}
      </Card>

      {/* Pagination */}
      {data && data.meta && data.meta.total > PAGE_SIZE && (
        <div className="flex items-center justify-between text-sm">
          <span className="text-foreground-muted">
            Page {page} of {totalPages} ({data.meta.total} events)
          </span>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1}
              className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs text-foreground transition-colors hover:bg-surface-elevated disabled:cursor-not-allowed disabled:opacity-50"
            >
              <ChevronLeft className="h-3.5 w-3.5" />
              Previous
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages}
              className="inline-flex items-center gap-1 rounded-md border border-border px-3 py-1.5 text-xs text-foreground transition-colors hover:bg-surface-elevated disabled:cursor-not-allowed disabled:opacity-50"
            >
              Next
              <ChevronRight className="h-3.5 w-3.5" />
            </button>
          </div>
        </div>
      )}
    </div>
  );
}

function Field({
  label,
  value,
  mono,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
}) {
  return (
    <div>
      <div className="text-foreground-muted">{label}</div>
      <div className={mono ? "break-all font-mono" : "font-medium text-foreground"}>
        {value}
      </div>
    </div>
  );
}
