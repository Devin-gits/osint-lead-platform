"use client";

import { cn } from "@/lib/utils/cn";
import { useState } from "react";
import { ChevronDown, ChevronRight } from "lucide-react";

export interface AuditLog {
  id: string;
  module: string;
  tool: string;
  checked_at: string;
  status: "ok" | "unknown" | "skipped";
  legal_basis: string;
  subject?: {
    email?: string;
    domain?: string;
    phone_redacted?: string;
    handle?: string;
    url?: string;
  };
  raw_stderr_json?: string;
}

export interface AuditLogPanelProps {
  events: AuditLog[];
  className?: string;
}

export function AuditLogPanel({ events, className }: AuditLogPanelProps) {
  const [expanded, setExpanded] = useState<Set<string>>(new Set());

  const toggle = (id: string) => {
    const next = new Set(expanded);
    if (next.has(id)) next.delete(id);
    else next.add(id);
    setExpanded(next);
  };

  return (
    <div className={cn("space-y-2", className)}>
      <h3 className="text-sm font-semibold text-foreground">Audit log</h3>
      {events.length === 0 ? (
        <p className="text-sm text-foreground-muted">No audit events yet.</p>
      ) : (
        <ul className="divide-y divide-border rounded-lg border border-border bg-surface">
          {events.map((event) => (
            <li key={event.id} className="px-4 py-3">
              <button
                onClick={() => toggle(event.id)}
                className="flex w-full items-center justify-between text-left focus:outline-none"
              >
                <span className="text-sm text-foreground-secondary">
                  <span className="font-medium text-foreground">{event.module}</span>
                  {event.tool && <span className="text-foreground-muted"> ({event.tool})</span>}
                  {" — "}{event.status}{" — "}{event.legal_basis}
                  {event.subject?.url && (
                    <span className="text-foreground-muted"> — {event.subject.url}</span>
                  )}
                </span>
                <span className="flex items-center gap-2">
                  {event.checked_at && (
                    <span className="hidden text-xs text-foreground-muted sm:inline">
                      {new Date(event.checked_at).toLocaleString()}
                    </span>
                  )}
                  {expanded.has(event.id) ? (
                    <ChevronDown className="h-4 w-4 text-foreground-muted" />
                  ) : (
                    <ChevronRight className="h-4 w-4 text-foreground-muted" />
                  )}
                </span>
              </button>
              {expanded.has(event.id) && event.raw_stderr_json && (
                <pre className="mt-2 overflow-x-auto rounded bg-surface-elevated p-2 text-xs text-foreground-muted">
                  {event.raw_stderr_json}
                </pre>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
