"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import { Box, Search, X } from "lucide-react";
import { useModules } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { Select } from "@/components/ui/Select";
import type { ModuleDevStatus, ModuleInfo } from "@/lib/api/types";

const statusOptions = [
  { value: "", label: "All statuses" },
  { value: "available", label: "Available" },
  { value: "in_development", label: "In development" },
  { value: "planned", label: "Planned" },
  { value: "not_configured", label: "Not configured" },
];

function variantForStatus(status: string): "success" | "warning" | "outline" | "muted" {
  switch (status) {
    case "available":
      return "success";
    case "in_development":
      return "warning";
    case "planned":
      return "outline";
    default:
      return "muted";
  }
}

function statusLabel(status: ModuleDevStatus): string {
  switch (status) {
    case "available":
      return "Available";
    case "in_development":
      return "In development";
    case "planned":
      return "Planned";
    case "not_configured":
      return "Not configured";
    default:
      return status;
  }
}

export default function ModulesPage() {
  const { data: modules, isLoading, error } = useModules();
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("");

  const filtered = useMemo(() => {
    if (!modules) return [];
    return modules.filter((m) => {
      if (statusFilter && m.dev_status !== statusFilter) return false;
      if (query) {
        const q = query.toLowerCase();
        return (
          m.name.toLowerCase().includes(q) ||
          m.display_name.toLowerCase().includes(q) ||
          m.description.toLowerCase().includes(q)
        );
      }
      return true;
    });
  }, [modules, query, statusFilter]);

  const grouped = useMemo(() => {
    const available: ModuleInfo[] = [];
    const inDevelopment: ModuleInfo[] = [];
    const planned: ModuleInfo[] = [];
    const notConfigured: ModuleInfo[] = [];
    filtered.forEach((m) => {
      if (m.dev_status === "available") available.push(m);
      else if (m.dev_status === "in_development") inDevelopment.push(m);
      else if (m.dev_status === "planned") planned.push(m);
      else notConfigured.push(m);
    });
    return { available, inDevelopment, planned, notConfigured };
  }, [filtered]);

  const hasFilters = query !== "" || statusFilter !== "";

  return (
    <div className="space-y-6">
      <PageHeader title="Modules" description="Module registry — status, configuration, and documentation." />

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-foreground-muted" />
          <input
            type="text"
            placeholder="Search modules..."
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
          className="w-full sm:w-48"
        />
      </div>

      {error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger" role="alert">
          Failed to load modules: {error.message}
        </div>
      )}

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3, 4].map((i) => (
            <Card key={i} className="h-40">
              <Skeleton className="h-5 w-2/3" />
              <Skeleton className="mt-3 h-4 w-full" />
              <Skeleton className="mt-2 h-4 w-1/2" />
            </Card>
          ))}
        </div>
      ) : modules?.length === 0 ? (
        <Card className="p-6">
          <EmptyState icon={Box} title="No modules" description="The module registry is empty." />
        </Card>
      ) : filtered.length === 0 && hasFilters ? (
        <Card className="p-6">
          <EmptyState
            icon={Search}
            title="No matching modules"
            description="Try a different search term or status filter."
          />
        </Card>
      ) : (
        <div className="space-y-8">
          {grouped.available.length > 0 && (
            <ModuleGroup title="Available" modules={grouped.available} />
          )}
          {grouped.inDevelopment.length > 0 && (
            <ModuleGroup title="In development" modules={grouped.inDevelopment} />
          )}
          {grouped.planned.length > 0 && (
            <ModuleGroup title="Planned" modules={grouped.planned} />
          )}
          {grouped.notConfigured.length > 0 && (
            <ModuleGroup title="Not configured" modules={grouped.notConfigured} />
          )}
        </div>
      )}
    </div>
  );
}

function ModuleGroup({ title, modules }: { title: string; modules: ModuleInfo[] }) {
  return (
    <section>
      <h2 className="mb-3 text-sm font-semibold text-foreground-muted">
        {title} ({modules.length})
      </h2>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {modules.map((module) => (
          <Link key={module.name} href={`/modules/${module.name}`} className="block">
            <Card className="h-full transition-colors hover:border-primary/50">
              <div className="flex items-start justify-between gap-2">
                <h3 className="font-medium text-foreground">{module.display_name}</h3>
                <Badge variant={variantForStatus(module.dev_status)} className="shrink-0">
                  {statusLabel(module.dev_status)}
                </Badge>
              </div>
              <p className="mt-2 text-sm text-foreground-muted line-clamp-2">
                {module.description}
              </p>
              <div className="mt-3 flex flex-wrap items-center gap-2 text-xs text-foreground-muted">
                <span className="rounded bg-surface-elevated px-1.5 py-0.5 capitalize">
                  {module.category}
                </span>
                <span className="rounded bg-surface-elevated px-1.5 py-0.5">
                  min: {module.min_input_field}
                </span>
              </div>
            </Card>
          </Link>
        ))}
      </div>
    </section>
  );
}
