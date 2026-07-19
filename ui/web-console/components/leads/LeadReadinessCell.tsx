"use client";

import { Mail, Phone, Globe, Users, FileSearch, Building2 } from "lucide-react";
import { StatusChip } from "@/components/ui/StatusChip";
import { cn } from "@/lib/utils/cn";
import type { LeadSummary, LeadRecord, ModuleStatus } from "@/lib/api/types";

type ReadinessLead = LeadSummary | LeadRecord;

const READINESS_KEYS = [
  { key: "email_validate", label: "Email", icon: Mail },
  { key: "phone_validate", label: "Phone", icon: Phone },
  { key: "domain_intel", label: "Domain", icon: Globe },
  { key: "social_footprint", label: "Social", icon: Users },
  { key: "extraction", label: "Extract", icon: FileSearch },
  { key: "company_enrich", label: "Company", icon: Building2 },
] as const;

function getStatus(
  lead: ReadinessLead,
  key: string
): ModuleStatus | undefined {
  const result = (lead as unknown as Record<string, { status?: ModuleStatus } | undefined>)[
    key
  ];
  return result?.status;
}

export interface LeadReadinessCellProps {
  lead: ReadinessLead;
  className?: string;
}

export function LeadReadinessCell({ lead, className }: LeadReadinessCellProps) {
  return (
    <div className={cn("flex flex-wrap items-center gap-2", className)}>
      {READINESS_KEYS.map(({ key, label, icon: Icon }) => {
        const status = getStatus(lead, key) ?? "not_run";
        return (
          <span
            key={key}
            className="inline-flex items-center gap-1.5 rounded-md border border-border bg-surface px-2 py-1 text-xs"
            title={`${label}: ${status.replace("_", " ")}`}
          >
            <Icon className="h-3.5 w-3.5 text-foreground-muted" aria-hidden="true" />
            <span className="sr-only">{label}</span>
            <StatusChip status={status} className="text-[10px]" />
          </span>
        );
      })}
    </div>
  );
}
