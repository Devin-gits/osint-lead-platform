import { cn } from "@/lib/utils/cn";
import { Badge } from "./Badge";

export type StatusChipStatus =
  | "ok"
  | "partial"
  | "unknown"
  | "skipped"
  | "pending"
  | "not_run"
  | "error";

export interface StatusChipProps {
  status: StatusChipStatus;
  className?: string;
}

export function StatusChip({ status, className }: StatusChipProps) {
  const labels: Record<StatusChipStatus, string> = {
    ok: "ok",
    partial: "partial",
    unknown: "unknown",
    skipped: "skipped",
    pending: "pending",
    not_run: "not run",
    error: "error",
  };

  const variants: Record<StatusChipStatus, Parameters<typeof Badge>[0]["variant"]> = {
    ok: "success",
    partial: "warning",
    unknown: "warning",
    skipped: "muted",
    pending: "primary",
    not_run: "outline",
    error: "danger",
  };

  return (
    <Badge variant={variants[status]} className={cn("capitalize", className)}>
      {labels[status]}
    </Badge>
  );
}
