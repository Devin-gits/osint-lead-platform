"use client";

import { useParams } from "next/navigation";
import Link from "next/link";
import { ArrowLeft, Box } from "lucide-react";
import { useModule } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Button } from "@/components/ui/Button";
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
            description={error?.message || "This module does not exist."}
          />
        </Card>
      </div>
    );
  }

  return (
    <div className="space-y-6">
      <PageHeader title={module.display_name} description={module.description}>
        <Link href="/modules">
          <Button variant="ghost" size="sm">
            <ArrowLeft className="mr-1.5 h-4 w-4" />
            Back to modules
          </Button>
        </Link>
      </PageHeader>

      <div className="grid gap-6 lg:grid-cols-3">
        <Card className="lg:col-span-2 space-y-4">
          <div className="flex items-center gap-2">
            <Badge variant={variantForStatus(module.dev_status)}>
              {module.dev_status.replace("_", " ")}
            </Badge>
            <Badge variant="secondary" className="capitalize">
              {module.category}
            </Badge>
          </div>

          <div className="grid gap-4 sm:grid-cols-2">
            <Field label="Name" value={module.name} />
            <Field label="Namespaced key" value={module.namespaced_key} />
            <Field label="Minimum input field" value={module.min_input_field} />
            <Field label="Risk note" value={module.risk_level_note} />
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

          {module.docs && (
            <div>
              <div className="text-xs text-foreground-muted">Documentation</div>
              <p className="mt-1 whitespace-pre-wrap text-sm text-foreground-secondary">
                {module.docs}
              </p>
            </div>
          )}
        </Card>

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
