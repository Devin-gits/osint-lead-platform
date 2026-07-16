import { cn } from "@/lib/utils/cn";
import { Badge } from "./Badge";

export type StatusChipStatus =
  | "ok"
  | "unknown"
  | "skipped"
  | "pending"
  | "not_run";

export interface StatusChipProps {
  status: StatusChipStatus;
  className?: string;
}

export function StatusChip({ status, className }: StatusChipProps) {
  const labels: Record<StatusChipStatus, string> = {
    ok: "ok",
    unknown: "unknown",
    skipped: "skipped",
    pending: "pending",
    not_run: "not run",
  };

  const variants: Record<StatusChipStatus, Parameters<typeof Badge>[0]["variant"]> = {
    ok: "success",
    unknown: "warning",
    skipped: "muted",
    pending: "primary",
    not_run: "outline",
  };

  return (
    <Badge variant={variants[status]} className={cn("capitalize", className)}>
      {labels[status]}
    </Badge>
  );
}
