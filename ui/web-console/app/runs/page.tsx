"use client";

import { Play } from "lucide-react";
import { useRuns } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/Table";

export default function RunsPage() {
  const { data, isLoading, error } = useRuns({ page_size: 50 });

  return (
    <div className="space-y-6">
      <PageHeader title="Runs" description="Pipeline run history and audit trails." />

      {error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger">
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
          <EmptyState icon={Play} title="No runs yet" description="Pipeline runs will appear here." />
        ) : (
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader>Type</TableHeader>
                <TableHeader>Status</TableHeader>
                <TableHeader>Started</TableHeader>
                <TableHeader>Leads</TableHeader>
                <TableHeader>Modules</TableHeader>
                <TableHeader>Legal basis</TableHeader>
                <TableHeader>Error</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              {data?.data.map((run) => (
                <TableRow key={run.id}>
                  <TableCell className="capitalize">{run.type}</TableCell>
                  <TableCell>
                    <Badge variant={variantForStatus(run.status)}>{run.status}</Badge>
                  </TableCell>
                  <TableCell>{new Date(run.started_at).toLocaleString()}</TableCell>
                  <TableCell>{run.lead_ids.length}</TableCell>
                  <TableCell>{run.modules_executed.join(", ")}</TableCell>
                  <TableCell>{run.legal_basis}</TableCell>
                  <TableCell>{run.error || "—"}</TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </Card>
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
