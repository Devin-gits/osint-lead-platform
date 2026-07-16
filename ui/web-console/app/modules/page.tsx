"use client";

import Link from "next/link";
import { Box } from "lucide-react";
import { useModules } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";

export default function ModulesPage() {
  const { data: modules, isLoading, error } = useModules();

  return (
    <div className="space-y-6">
      <PageHeader title="Modules" description="Module status, configuration, and documentation." />

      {error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger">
          Failed to load modules: {error.message}
        </div>
      )}

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[1, 2, 3].map((i) => (
            <Card key={i} className="h-40">
              <Skeleton className="h-6 w-1/2" />
            </Card>
          ))}
        </div>
      ) : modules?.length === 0 ? (
        <Card>
          <EmptyState icon={Box} title="No modules" description="The module registry is empty." />
        </Card>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {modules?.map((module) => (
            <Link key={module.name} href={`/modules/${module.name}`} className="block">
              <Card className="h-full transition-colors hover:border-primary/50">
                <div className="flex items-start justify-between">
                  <div>
                    <h3 className="font-medium text-foreground">{module.display_name}</h3>
                    <p className="mt-1 text-sm text-foreground-muted line-clamp-2">{module.description}</p>
                  </div>
                  <Badge variant={variantForStatus(module.dev_status)}>
                    {module.dev_status.replace("_", " ")}
                  </Badge>
                </div>
                <div className="mt-4 flex flex-wrap gap-2 text-xs text-foreground-muted">
                  <span className="capitalize">{module.category}</span>
                  <span>•</span>
                  <span>min: {module.min_input_field}</span>
                </div>
              </Card>
            </Link>
          ))}
        </div>
      )}
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
