"use client";

import { Menu } from "lucide-react";
import { IconButton } from "@/components/ui/IconButton";
import { usePathname } from "next/navigation";

export interface TopBarProps {
  onMenuClick: () => void;
  menuButtonRef?: React.RefObject<HTMLButtonElement | null>;
}

const routeTitles: Record<string, string> = {
  "/command-center": "Command Center",
  "/leads": "Leads",
  "/runs": "Runs",
  "/modules": "Modules",
  "/compliance": "Compliance",
  "/settings": "Settings",
  "/style-guide": "Style guide",
};

function pageTitle(pathname: string): string {
  if (routeTitles[pathname]) return routeTitles[pathname];
  if (pathname.startsWith("/leads/")) return "Lead detail";
  if (pathname.startsWith("/runs/")) return "Run detail";
  if (pathname.startsWith("/modules/")) return "Module detail";
  return "OSINT Lead Console";
}

export function TopBar({ onMenuClick, menuButtonRef }: TopBarProps) {
  const pathname = usePathname();
  const title = pageTitle(pathname);

  return (
    <header className="sticky top-0 z-30 flex h-14 shrink-0 items-center justify-between border-b border-border bg-surface/95 px-4 backdrop-blur md:px-6 lg:px-8">
      <div className="flex min-w-0 items-center gap-3">
        <IconButton
          ref={menuButtonRef}
          icon={Menu}
          label="Open navigation"
          onClick={onMenuClick}
          className="lg:hidden"
        />
        <span className="text-base font-semibold tracking-tight text-foreground md:text-lg">
          {title}
        </span>
      </div>
    </header>
  );
}
