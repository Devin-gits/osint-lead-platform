"use client";

import { Shield, XCircle, AlertTriangle, CheckCircle2 } from "lucide-react";
import { useComplianceSummary } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";

export default function CompliancePage() {
  const { data, isLoading, error } = useComplianceSummary();

  return (
    <div className="space-y-6">
      <PageHeader title="Compliance" description="Hard rules, risk table, and pre-run checklist." />

      {error && (
        <div className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger">
          Failed to load compliance summary: {error.message}
        </div>
      )}

      {isLoading ? (
        <div className="space-y-6">
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
        </div>
      ) : !data ? (
        <Card>
          <EmptyState icon={Shield} title="No compliance data" description="Compliance summary is unavailable." />
        </Card>
      ) : (
        <>
          <Card>
            <h2 className="mb-4 flex items-center gap-2 text-base font-semibold text-foreground">
              <AlertTriangle className="h-5 w-5 text-warning" />
              Hard rules
            </h2>
            <ul className="space-y-4">
              {data.hard_rules.map((rule) => (
                <li key={rule.id} className="rounded-md border border-border bg-surface p-4">
                  <div className="font-medium text-foreground">
                    {rule.id}. {rule.title}
                  </div>
                  <p className="mt-1 text-sm text-foreground-muted">{rule.summary}</p>
                </li>
              ))}
            </ul>
          </Card>

          <div className="grid gap-6 lg:grid-cols-2">
            <Card>
              <h2 className="mb-4 text-base font-semibold text-foreground">Risk table</h2>
              <div className="overflow-x-auto">
                <table className="w-full text-left text-sm">
                  <thead className="border-b border-border text-foreground-muted">
                    <tr>
                      <th className="pb-2 font-medium">Category</th>
                      <th className="pb-2 font-medium">Risk</th>
                      <th className="pb-2 font-medium">Notes</th>
                    </tr>
                  </thead>
                  <tbody className="divide-y divide-border">
                    {data.risk_table.map((row, i) => (
                      <tr key={i}>
                        <td className="py-3 text-foreground">{row.category}</td>
                        <td className="py-3">{row.risk_level}</td>
                        <td className="py-3 text-foreground-muted">{row.notes}</td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </Card>

            <Card>
              <h2 className="mb-4 flex items-center gap-2 text-base font-semibold text-foreground">
                <CheckCircle2 className="h-5 w-5 text-success" />
                Pre-run checklist
              </h2>
              <ul className="space-y-2">
                {data.checklist.map((item) => (
                  <li key={item.id} className="flex items-start gap-3 text-sm text-foreground-secondary">
                    <CheckCircle2 className="mt-0.5 h-4 w-4 flex-shrink-0 text-success" />
                    {item.label}
                  </li>
                ))}
              </ul>

              <h3 className="mb-3 mt-6 flex items-center gap-2 text-sm font-semibold text-foreground">
                <XCircle className="h-4 w-4 text-danger" />
                Exclusions
              </h3>
              <ul className="space-y-1">
                {data.exclusions.map((exclusion, i) => (
                  <li key={i} className="text-sm text-danger">
                    {exclusion}
                  </li>
                ))}
              </ul>
            </Card>
          </div>
        </>
      )}
    </div>
  );
}
