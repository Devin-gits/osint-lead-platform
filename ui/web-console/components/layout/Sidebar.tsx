"use client";

import { cn } from "@/lib/utils/cn";
import { usePathname } from "next/navigation";
import Link from "next/link";
import { Users, Box, Play, Shield, Settings, X } from "lucide-react";
import { IconButton } from "@/components/ui/IconButton";

export interface SidebarProps {
  open: boolean;
  onClose: () => void;
}

const navItems = [
  { href: "/leads", label: "Leads", icon: Users },
  { href: "/modules", label: "Modules", icon: Box },
  { href: "/runs", label: "Runs", icon: Play },
  { href: "/compliance", label: "Compliance", icon: Shield },
  { href: "/settings", label: "Settings", icon: Settings },
];

export function Sidebar({ open, onClose }: SidebarProps) {
  const pathname = usePathname();

  return (
    <>
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}
      <aside
        className={cn(
          "fixed left-0 top-0 z-50 h-full w-60 transform border-r border-border bg-surface pt-14 transition-transform duration-200 lg:static lg:transform-none lg:pt-0",
          open ? "translate-x-0" : "-translate-x-full lg:hidden"
        )}
      >
        <div className="flex h-14 items-center justify-between px-4 lg:hidden">
          <span className="font-semibold text-foreground">Menu</span>
          <IconButton icon={X} label="Close navigation" onClick={onClose} />
        </div>
        <nav className="px-3 py-4">
          <ul className="space-y-1">
            {navItems.map((item) => {
              const active = pathname === item.href || pathname.startsWith(`${item.href}/`);
              const Icon = item.icon;
              return (
                <li key={item.href}>
                  <Link
                    href={item.href}
                    onClick={onClose}
                    className={cn(
                      "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                      active
                        ? "bg-primary/10 text-primary"
                        : "text-foreground-secondary hover:bg-surface-elevated hover:text-foreground"
                    )}
                    aria-current={active ? "page" : undefined}
                  >
                    <Icon className="h-4 w-4" aria-hidden="true" />
                    {item.label}
                  </Link>
                </li>
              );
            })}
          </ul>
        </nav>
      </aside>
    </>
  );
}
