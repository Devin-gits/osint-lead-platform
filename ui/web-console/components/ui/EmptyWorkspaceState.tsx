import type { LucideIcon } from "lucide-react";
import Link from "next/link";
import { cn } from "@/lib/utils/cn";

interface ActionProps {
  label: string;
  href: string;
  variant?: "primary" | "secondary" | "ghost";
}

export interface EmptyWorkspaceStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  primaryAction?: ActionProps;
  secondaryAction?: ActionProps;
  children?: React.ReactNode;
  className?: string;
}

function ActionLink({ label, href, variant = "primary" }: ActionProps) {
  return (
    <Link
      href={href}
      className={cn(
        "inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2",
        variant === "primary" &&
          "bg-primary text-background hover:brightness-110 focus:ring-primary/50",
        variant === "secondary" &&
          "border border-border bg-surface text-foreground hover:bg-surface-elevated focus:ring-primary/50",
        variant === "ghost" &&
          "bg-transparent text-foreground hover:bg-surface-elevated focus:ring-primary/50"
      )}
    >
      {label}
    </Link>
  );
}

export function EmptyWorkspaceState({
  icon: Icon,
  title,
  description,
  primaryAction,
  secondaryAction,
  children,
  className,
}: EmptyWorkspaceStateProps) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center rounded-lg border border-border bg-surface px-6 py-16 text-center",
        className
      )}
      role="status"
      aria-live="polite"
    >
      {Icon && (
        <Icon
          className="mb-4 h-12 w-12 text-foreground-muted"
          aria-hidden="true"
        />
      )}
      <h3 className="text-lg font-semibold text-foreground">{title}</h3>
      {description && (
        <p className="mt-2 max-w-md text-sm text-foreground-secondary">
          {description}
        </p>
      )}
      {children && <div className="mt-6">{children}</div>}
      {!children && (primaryAction || secondaryAction) && (
        <div className="mt-6 flex flex-wrap items-center justify-center gap-3">
          {primaryAction && <ActionLink {...primaryAction} variant="primary" />}
          {secondaryAction && (
            <ActionLink {...secondaryAction} variant="secondary" />
          )}
        </div>
      )}
    </div>
  );
}
