"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { Plus, Search, Users } from "lucide-react";
import { useLeads, useCreateLead } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Dialog } from "@/components/ui/Dialog";
import { Badge } from "@/components/ui/Badge";
import { StatusChip } from "@/components/ui/StatusChip";
import { EmptyState } from "@/components/ui/EmptyState";
import { Skeleton } from "@/components/ui/Skeleton";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/Table";
import { cn } from "@/lib/utils/cn";
import { RiskLevel } from "@/lib/api/types";

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
  { value: "unknown", label: "unknown" },
  { value: "skipped", label: "skipped" },
  { value: "not_run", label: "not run" },
];

export default function LeadsPage() {
  const router = useRouter();
  const [filters, setFilters] = useState({
    stage: "",
    risk: "",
    module_status: "",
    q: "",
    page: 1,
    page_size: 25,
  });
  const [createOpen, setCreateOpen] = useState(false);
  const [form, setForm] = useState({
    name: "",
    email: "",
    phone: "",
    company: "",
    domain: "",
    permission_ref: "",
  });

  const { data, isLoading, error } = useLeads(filters);
  const create = useCreateLead();

  const handleCreate = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      const created = await create.mutateAsync({
        ...form,
        source_id: "",
      });
      setCreateOpen(false);
      setForm({ name: "", email: "", phone: "", company: "", domain: "", permission_ref: "" });
      router.push(`/leads/${created.id}`);
    } catch {
      // mutation error surfaces via create.error
    }
  };

  const stages = ["raw", "enriched", "validated", "crm_ready"];
  const countsByStage: Record<string, number> = {};
  data?.data.forEach((lead) => {
    countsByStage[lead.stage] = (countsByStage[lead.stage] || 0) + 1;
  });

  return (
    <div className="space-y-6">
      <PageHeader title="Leads" description="Manage and inspect enriched leads.">
        <Button onClick={() => setCreateOpen(true)} size="sm">
          <Plus className="mr-1.5 h-4 w-4" />
          Create lead
        </Button>
      </PageHeader>

      <Card className="p-4">
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          <Select
            label="Stage"
            options={stageOptions}
            value={filters.stage}
            onChange={(e) => setFilters({ ...filters, stage: e.target.value, page: 1 })}
          />
          <Select
            label="Risk level"
            options={riskOptions}
            value={filters.risk}
            onChange={(e) => setFilters({ ...filters, risk: e.target.value, page: 1 })}
          />
          <Select
            label="Module status"
            options={moduleStatusOptions}
            value={filters.module_status}
            onChange={(e) => setFilters({ ...filters, module_status: e.target.value, page: 1 })}
          />
          <div className="relative">
            <Input
              label="Search"
              placeholder="Name, email, company, domain"
              value={filters.q}
              onChange={(e) => setFilters({ ...filters, q: e.target.value, page: 1 })}
            />
            <Search className="pointer-events-none absolute right-3 top-8 h-4 w-4 text-foreground-muted" />
          </div>
        </div>
      </Card>

      <Card className="p-4">
        <h3 className="mb-3 text-sm font-semibold text-foreground">
          Stage funnel <span className="font-normal text-foreground-muted">(on this page)</span>
        </h3>
        <div className="flex flex-wrap gap-2">
          {stages.map((stage) => {
            const count = countsByStage[stage] || 0;
            const active = filters.stage === stage;
            return (
              <button
                key={stage}
                onClick={() => setFilters({ ...filters, stage: active ? "" : stage, page: 1 })}
                className={cn(
                  "flex items-center gap-2 rounded-md border px-3 py-1.5 text-sm transition-colors",
                  active
                    ? "border-primary bg-primary/10 text-primary"
                    : "border-border bg-surface text-foreground-secondary hover:bg-surface-elevated"
                )}
              >
                <span className="capitalize">{stage.replace("_", " ")}</span>
                <Badge variant={active ? "primary" : "outline"}>{count}</Badge>
              </button>
            );
          })}
        </div>
      </Card>

      {error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger">
          Failed to load leads: {error.message}
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
            icon={Users}
            title="No leads yet"
            description="Create a lead to start enrichment."
          />
        ) : (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader>Contact</TableHeader>
                <TableHeader>Company</TableHeader>
                <TableHeader>Stage</TableHeader>
                <TableHeader>Risk</TableHeader>
                <TableHeader>Modules</TableHeader>
                <TableHeader>Permission ref</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {data?.data.map((lead) => (
                <TableRow
                  key={lead.id}
                  onClick={() => router.push(`/leads/${lead.id}`)}
                  className="cursor-pointer"
                >
                  <TableCell>
                    <div className="font-medium text-foreground">{lead.name || "—"}</div>
                    <div className="text-xs text-foreground-muted">{lead.email || lead.domain || lead.id}</div>
                  </TableCell>
                  <TableCell>{lead.company || "—"}</TableCell>
                  <TableCell>
                    <Badge variant="secondary" className="capitalize">
                      {lead.stage.replace("_", " ")}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <RiskBadge level={lead.risk_level} />
                  </TableCell>
                  <TableCell>
                    <div className="flex flex-wrap gap-1">
                      {["email_validate", "phone_validate", "domain_intel", "social_footprint"].map((key) => {
                        const result = (lead as Record<string, unknown>)[key] as { status?: string } | undefined;
                        const status = result?.status || "not_run";
                        return (
                          <StatusChip
                            key={key}
                            status={status as "ok" | "unknown" | "skipped" | "pending" | "not_run"}
                            className="text-[10px]"
                          />
                        );
                      })}
                    </div>
                  </TableCell>
                  <TableCell>{lead.permission_ref || "—"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>

      {data?.meta && data.meta.total > 0 && (
        <div className="flex items-center justify-between text-sm text-foreground-muted">
          <span>
            Showing {(data.meta.page - 1) * data.meta.page_size + 1} -{" "}
            {Math.min(data.meta.page * data.meta.page_size, data.meta.total)} of {data.meta.total}
          </span>
          <div className="flex gap-2">
            <Button
              size="sm"
              variant="ghost"
              disabled={data.meta.page <= 1}
              onClick={() => setFilters({ ...filters, page: data.meta.page - 1 })}
            >
              Previous
            </Button>
            <Button
              size="sm"
              variant="ghost"
              disabled={data.meta.page * data.meta.page_size >= data.meta.total}
              onClick={() => setFilters({ ...filters, page: data.meta.page + 1 })}
            >
              Next
            </Button>
          </div>
        </div>
      )}

      <Dialog open={createOpen} onClose={() => setCreateOpen(false)} title="Create lead">
        <form onSubmit={handleCreate} className="space-y-4">
          <Input label="Name" value={form.name} onChange={(e) => setForm({ ...form, name: e.target.value })} />
          <Input
            label="Email"
            type="email"
            value={form.email}
            onChange={(e) => setForm({ ...form, email: e.target.value })}
          />
          <Input label="Phone" value={form.phone} onChange={(e) => setForm({ ...form, phone: e.target.value })} />
          <Input label="Company" value={form.company} onChange={(e) => setForm({ ...form, company: e.target.value })} />
          <Input label="Domain" value={form.domain} onChange={(e) => setForm({ ...form, domain: e.target.value })} />
          <Input
            label="Permission ref"
            value={form.permission_ref}
            onChange={(e) => setForm({ ...form, permission_ref: e.target.value })}
          />
          {create.error && (
            <div className="text-sm text-danger">{create.error.message}</div>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="ghost" onClick={() => setCreateOpen(false)}>
              Cancel
            </Button>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending ? "Creating…" : "Create"}
            </Button>
          </div>
        </form>
      </Dialog>
    </div>
  );
}

function RiskBadge({ level }: { level: RiskLevel }) {
  const variant =
    level === "low" ? "success" : level === "medium" ? "warning" : level === "high" ? "danger" : "outline";
  return <Badge variant={variant}>{level}</Badge>;
}
