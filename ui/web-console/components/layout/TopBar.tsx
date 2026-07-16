"use client";

import { Menu, Search, User } from "lucide-react";
import { IconButton } from "@/components/ui/IconButton";
import { Badge } from "@/components/ui/Badge";
import { useApiHealth } from "@/lib/api/hooks";

export interface TopBarProps {
  onMenuClick: () => void;
}

export function TopBar({ onMenuClick }: TopBarProps) {
  const { isLoading, isError } = useApiHealth();

  const badgeVariant = isLoading ? "warning" : isError ? "danger" : "success";
  const badgeText = isLoading ? "API checking…" : isError ? "API offline" : "Live API";

  return (
    <header className="flex h-14 items-center justify-between border-b border-border bg-surface px-4">
      <div className="flex items-center gap-3">
        <IconButton icon={Menu} label="Open navigation" onClick={onMenuClick} className="lg:hidden" />
        <span className="text-lg font-semibold tracking-tight text-foreground">
          OSINT Lead Console
        </span>
      </div>

      <div className="flex items-center gap-4">
        <Badge variant={badgeVariant}>{badgeText}</Badge>

        <div className="hidden items-center gap-2 rounded-md border border-border bg-background px-3 py-1.5 sm:flex">
          <Search className="h-4 w-4 text-foreground-muted" aria-hidden="true" />
          <span className="text-sm text-foreground-muted">Search (stub)</span>
        </div>

        <div className="flex h-8 w-8 items-center justify-center rounded-full border border-border bg-surface-elevated text-foreground-muted">
          <User className="h-4 w-4" aria-hidden="true" />
        </div>
      </div>
    </header>
  );
}
