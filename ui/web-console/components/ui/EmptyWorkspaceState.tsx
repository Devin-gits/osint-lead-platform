import { cn } from "@/lib/utils/cn";
import { LucideIcon } from "lucide-react";

export interface EmptyWorkspaceStateProps {
  icon?: LucideIcon;
  title: string;
  description?: string;
  children?: React.ReactNode;
  className?: string;
}

export function EmptyWorkspaceState({
  icon: Icon,
  title,
  description,
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
    </div>
  );
}
