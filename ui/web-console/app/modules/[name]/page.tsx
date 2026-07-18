"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Box, Info } from "lucide-react";
import { useModule } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";

export default function ModuleDetailPage() {
  const { name } = useParams<{ name: string }>();
  const { data: module, isLoading, error } = useModule(name);

  if (isLoading) {
    return (
      <div className="space-y-6">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-64 w-full" />
      </div>
    );
  }

  if (error || !module) {
    return (
      <div className="space-y-6">
        <PageHeader title="Module not found" />
        <Card className="p-6">
          <EmptyState
            icon={Box}
            title="Failed to load module"
            description={error?.message || "This module does not exist or the API is unreachable."}
          />
        </Card>
      </div>
    );
  }

  const isAvailable = module.dev_status === "available";

  return (
    <div className="space-y-6">
      <PageHeader title={module.display_name} description={module.description}>
        <Link
          href="/modules"
          className="inline-flex items-center rounded-md bg-transparent px-2.5 py-1 text-xs font-medium text-foreground transition-colors hover:bg-surface-elevated focus:outline-none focus:ring-2 focus:ring-primary/50"
        >
          <ArrowLeft className="mr-1.5 h-4 w-4" />
          Back to modules
        </Link>
      </PageHeader>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2 space-y-5">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant={variantForStatus(module.dev_status)}>
              {statusLabel(module.dev_status)}
            </Badge>
            <Badge variant="secondary" className="capitalize">
              {module.category}
            </Badge>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Module name" value={module.name} />
            <Field label="Namespaced key" value={module.namespaced_key} />
            <Field label="Minimum input field" value={module.min_input_field} />
            <Field label="Risk note" value={module.risk_level_note || "—"} />
          </div>

          <div>
            <div className="text-xs text-foreground-muted">Backing tools</div>
            <ul className="mt-1 list-inside list-disc text-sm text-foreground-secondary">
              {module.backing_tools.length === 0 ? (
                <li>None listed</li>
              ) : (
                module.backing_tools.map((tool) => <li key={tool}>{tool}</li>)
              )}
            </ul>
          </div>

          {isAvailable && (
            <div className="flex items-start gap-2 rounded-md border border-primary/20 bg-primary/5 p-3 text-sm text-foreground-secondary">
              <Info className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
              <span>
                To run this module, open a lead&apos;s detail page or use the{" "}
                <Link href="/leads/create" className="font-medium text-primary hover:underline">
                  create lead flow
                </Link>
                .
              </span>
            </div>
          )}
        </Card>

        <div className="space-y-6">
          <Card>
            <h3 className="mb-3 text-sm font-semibold text-foreground">Configuration</h3>
            {module.config_schema && module.config_schema.length > 0 ? (
              <ul className="space-y-2">
                {module.config_schema.map((field) => (
                  <li key={field.key} className="text-sm">
                    <span className="font-medium text-foreground">{field.label}</span>
                    <span className="text-foreground-muted"> ({field.type})</span>
                    {field.required && <span className="text-danger"> *</span>}
                  </li>
                ))}
              </ul>
            ) : (
              <p className="text-sm text-foreground-muted">No configuration schema exposed yet.</p>
            )}
          </Card>

          <Card>
            <h3 className="mb-3 text-sm font-semibold text-foreground">Links</h3>
            <div className="space-y-2 text-sm">
              <Link
                href="/leads"
                className="block text-primary hover:underline"
              >
                View related leads
              </Link>
            </div>
          </Card>
        </div>
      </div>

      <Card className="space-y-3">
        <h3 className="text-sm font-semibold text-foreground">Documentation</h3>
        {module.docs ? (
          <div className="whitespace-pre-wrap text-sm text-foreground-secondary">
            {module.docs}
          </div>
        ) : (
          <p className="text-sm text-foreground-muted">
            No documentation payload from control-plane yet.
          </p>
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

function variantForStatus(status: string) {
  switch (status) {
    case "available":
      return "success" as const;
    case "in_development":
      return "warning" as const;
    case "planned":
      return "outline" as const;
    default:
      return "muted" as const;
  }
}

function statusLabel(status: string): string {
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
