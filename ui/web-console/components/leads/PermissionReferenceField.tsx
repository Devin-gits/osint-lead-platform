"use client";

import { useId } from "react";
import { Input } from "@/components/ui/Input";
import { cn } from "@/lib/utils/cn";

export interface PermissionReferenceFieldProps {
  value: string;
  onChange: (value: string) => void;
  error?: string;
  disabled?: boolean;
}

export function PermissionReferenceField({
  value,
  onChange,
  error,
  disabled,
}: PermissionReferenceFieldProps) {
  const id = useId();
  const inputId = `permission-ref-${id}`;
  const errorId = `permission-ref-error-${id}`;

  return (
    <div className="space-y-2">
      <label htmlFor={inputId} className="block text-sm text-foreground-secondary">
        Permission reference
        <span className="ml-1 text-danger" aria-hidden="true">*</span>
      </label>
      <Input
        id={inputId}
        placeholder="e.g. cmp-2026-001 / GDPR Art.6(1)(f) campaign"
        value={value}
        onChange={(e) => onChange(e.target.value)}
        disabled={disabled}
        aria-required="true"
        aria-invalid={!!error}
        aria-describedby={error ? errorId : undefined}
      />
      <p className="text-xs text-foreground-muted">
        Required for extraction / compliance. Documents the lawful basis /
        campaign authorization for processing this lead. The control-plane
        rejects extraction runs without it.
      </p>
      {error && (
        <p id={errorId} className="text-sm text-danger" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}

export function PermissionReferenceInline({
  permissionRef,
  className,
}: {
  permissionRef?: string;
  className?: string;
}) {
  if (permissionRef) {
    return (
      <span
        className={cn("text-sm text-foreground-secondary", className)}
        title={permissionRef}
      >
        {permissionRef}
      </span>
    );
  }
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full border border-warning/20 bg-warning/10 px-2 py-0.5 text-xs font-medium text-warning",
        className
      )}
    >
      Missing
    </span>
  );
}
