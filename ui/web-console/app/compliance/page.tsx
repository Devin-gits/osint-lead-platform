"use client";

import { useState } from "react";
import Link from "next/link";
import {
  Shield,
  XCircle,
  AlertTriangle,
  CheckCircle2,
  Info,
} from "lucide-react";
import { useComplianceSummary } from "@/lib/api/hooks";
import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { Badge } from "@/components/ui/Badge";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";

function riskVariant(level: string) {
  switch (level.toLowerCase()) {
    case "low":
      return "success" as const;
    case "medium":
      return "warning" as const;
    case "high":
      return "danger" as const;
    default:
      return "muted" as const;
  }
}

export default function CompliancePage() {
  const { data, isLoading, error } = useComplianceSummary();
  const [checked, setChecked] = useState<Set<string>>(new Set());

  const toggle = (id: string) => {
    setChecked((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };

  return (
    <div className="space-y-6">
      <PageHeader
        title="Compliance"
        description="GDPR Art.6(1)(f) legitimate interest posture for permitted lead processing."
      />

      <div className="flex items-start gap-2 rounded-md border border-primary/20 bg-primary/5 p-3 text-sm text-foreground-secondary">
        <Info className="mt-0.5 h-4 w-4 shrink-0 text-primary" />
        <span>
          This workspace reflects the platform&apos;s compliance configuration from the control-plane API.
          It is not legal advice. Consult your DPO for binding guidance.
        </span>
      </div>

      {error && (
        <div
          className="rounded-md border border-danger/20 bg-danger/10 p-4 text-sm text-danger"
          role="alert"
        >
          Failed to load compliance summary: {error.message}
        </div>
      )}

      {isLoading ? (
        <div className="space-y-6">
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-40 w-full" />
          <Skeleton className="h-32 w-full" />
        </div>
      ) : !data ? (
        <Card className="p-6">
          <EmptyState
            icon={Shield}
            title="No compliance data"
            description="Compliance summary is unavailable from the control-plane."
          />
        </Card>
      ) : (
        <>
          {/* Hard rules */}
          <Card>
            <h2 className="mb-4 flex items-center gap-2 text-base font-semibold text-foreground">
              <AlertTriangle className="h-5 w-5 text-warning" />
              Hard rules
            </h2>
            {data.hard_rules.length === 0 ? (
              <p className="text-sm text-foreground-muted">No hard rules in payload.</p>
            ) : (
              <ul className="space-y-3">
                {data.hard_rules.map((rule) => (
                  <li
                    key={rule.id}
                    className="rounded-md border border-border bg-surface-elevated p-4"
                  >
                    <div className="font-medium text-foreground">
                      {rule.id}. {rule.title}
                    </div>
                    <p className="mt-1 text-sm text-foreground-muted">{rule.summary}</p>
                  </li>
                ))}
              </ul>
            )}
          </Card>

          {/* Risk table + Checklist grid */}
          <div className="grid gap-6 lg:grid-cols-2">
            {/* Risk table */}
            <Card>
              <h2 className="mb-4 text-base font-semibold text-foreground">
                Risk table
              </h2>
              {data.risk_table.length === 0 ? (
                <p className="text-sm text-foreground-muted">No risk categories in payload.</p>
              ) : (
                <div className="overflow-x-auto">
                  <table className="w-full text-left text-sm">
                    <thead className="border-b border-border text-foreground-muted">
                      <tr>
                        <th className="pb-2 pr-4 font-medium">Category</th>
                        <th className="pb-2 pr-4 font-medium">Risk level</th>
                        <th className="pb-2 font-medium">Notes</th>
                      </tr>
                    </thead>
                    <tbody className="divide-y divide-border">
                      {data.risk_table.map((row, i) => (
                        <tr key={i}>
                          <td className="py-3 pr-4 text-foreground">{row.category}</td>
                          <td className="py-3 pr-4">
                            <Badge variant={riskVariant(row.risk_level)}>
                              {row.risk_level}
                            </Badge>
                          </td>
                          <td className="py-3 text-foreground-muted">{row.notes}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>
              )}
            </Card>

            {/* Pre-run checklist */}
            <Card>
              <h2 className="mb-4 flex items-center gap-2 text-base font-semibold text-foreground">
                <CheckCircle2 className="h-5 w-5 text-success" />
                Pre-run checklist
              </h2>
              {data.checklist.length === 0 ? (
                <p className="text-sm text-foreground-muted">No checklist items in payload.</p>
              ) : (
                <ul className="space-y-2">
                  {data.checklist.map((item) => (
                    <li key={item.id}>
                      <label className="flex cursor-pointer items-start gap-3 rounded-md p-2 transition-colors hover:bg-surface-elevated">
                        <input
                          type="checkbox"
                          checked={checked.has(item.id)}
                          onChange={() => toggle(item.id)}
                          className="mt-0.5 h-4 w-4 shrink-0 rounded border-border text-primary accent-primary focus:ring-primary/50"
                        />
                        <span className="text-sm text-foreground-secondary">{item.label}</span>
                      </label>
                    </li>
                  ))}
                </ul>
              )}
              <p className="mt-3 text-xs text-foreground-muted">
                Checkboxes are local only and not persisted to the API.
              </p>
            </Card>
          </div>

          {/* Exclusions */}
          <Card>
            <h2 className="mb-4 flex items-center gap-2 text-base font-semibold text-foreground">
              <XCircle className="h-5 w-5 text-danger" />
              Exclusions (out of scope)
            </h2>
            {data.exclusions.length === 0 ? (
              <p className="text-sm text-foreground-muted">No exclusions listed.</p>
            ) : (
              <ul className="space-y-2">
                {data.exclusions.map((exclusion, i) => (
                  <li
                    key={i}
                    className="flex items-center gap-2 text-sm text-foreground-secondary"
                  >
                    <XCircle className="h-3.5 w-3.5 shrink-0 text-danger" />
                    {exclusion}
                  </li>
                ))}
              </ul>
            )}
          </Card>

          {/* Links */}
          <Card>
            <h3 className="mb-3 text-sm font-semibold text-foreground">Related</h3>
            <div className="flex flex-wrap gap-4 text-sm">
              <Link href="/modules" className="text-primary hover:underline">
                Modules registry
              </Link>
              <Link href="/leads" className="text-primary hover:underline">
                Leads queue
              </Link>
            </div>
          </Card>
        </>
      )}
    </div>
  );
}
