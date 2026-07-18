"use client";

import { useEffect, useRef, useCallback } from "react";
import { cn } from "@/lib/utils/cn";
import { usePathname } from "next/navigation";
import Link from "next/link";
import {
  LayoutDashboard,
  Users,
  Play,
  ScrollText,
  Box,
  Shield,
  Settings,
  X,
  LucideIcon,
} from "lucide-react";
import { IconButton } from "@/components/ui/IconButton";

export interface ResponsiveSidebarProps {
  open: boolean;
  onClose: () => void;
  triggerRef?: React.RefObject<HTMLButtonElement | null>;
}

type NavItem = {
  href: string;
  label: string;
  icon: LucideIcon;
  comingSoon?: boolean;
};

type NavGroup = {
  label: string;
  items: NavItem[];
};

const groups: NavGroup[] = [
  {
    label: "Workspace",
    items: [
      { href: "/command-center", label: "Command Center", icon: LayoutDashboard },
      { href: "/leads", label: "Leads", icon: Users },
      { href: "/runs", label: "Runs", icon: Play },
      { href: "/audit", label: "Audit Log", icon: ScrollText, comingSoon: true },
    ],
  },
  {
    label: "Operations",
    items: [
      { href: "/modules", label: "Modules", icon: Box },
      { href: "/compliance", label: "Compliance", icon: Shield },
    ],
  },
  {
    label: "Administration",
    items: [{ href: "/settings", label: "Settings", icon: Settings }],
  },
];

function isActive(pathname: string, href: string): boolean {
  if (pathname === href) return true;
  if (href === "/command-center") return false;
  if (href === "/" && pathname === "/command-center") return true;
  return pathname.startsWith(`${href}/`);
}

function getFocusable(container: HTMLElement): HTMLElement[] {
  return Array.from(
    container.querySelectorAll<HTMLElement>(
      'a[href], button, input, textarea, select, details, [tabindex]:not([tabindex="-1"])'
    )
  ).filter((el) => !el.hasAttribute("disabled") && !el.getAttribute("aria-disabled"));
}

export function ResponsiveSidebar({ open, onClose, triggerRef }: ResponsiveSidebarProps) {
  const pathname = usePathname();
  const drawerRef = useRef<HTMLElement>(null);
  const closeButtonRef = useRef<HTMLButtonElement>(null);
  const wasOpen = useRef(false);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
        return;
      }
      if (e.key !== "Tab" || !drawerRef.current) return;
      const focusable = getFocusable(drawerRef.current);
      if (focusable.length === 0) return;
      const first = focusable[0];
      const last = focusable[focusable.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    },
    [onClose]
  );

  useEffect(() => {
    if (!open) return;
    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const timer = setTimeout(
      () => {
        closeButtonRef.current?.focus();
      },
      prefersReducedMotion ? 0 : 210
    );
    document.addEventListener("keydown", handleKeyDown);
    return () => {
      clearTimeout(timer);
      document.removeEventListener("keydown", handleKeyDown);
    };
  }, [open, handleKeyDown]);

  useEffect(() => {
    const closing = wasOpen.current && !open;
    wasOpen.current = open;
    if (!closing) return;
    const prefersReducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const timer = setTimeout(
      () => {
        const trigger = triggerRef?.current;
        if (trigger && trigger.offsetParent !== null) {
          trigger.focus();
        }
      },
      prefersReducedMotion ? 0 : 210
    );
    return () => clearTimeout(timer);
  }, [open, triggerRef]);

  const renderItem = (item: NavItem) => {
    const active = isActive(pathname, item.href);
    const Icon = item.icon;
    const content = (
      <>
        <Icon className="h-5 w-5 shrink-0" aria-hidden="true" />
        <span className="truncate lg:hidden xl:inline">{item.label}</span>
        <span
          className={cn(
            "pointer-events-none absolute left-full top-1/2 z-50 ml-2 -translate-y-1/2 whitespace-nowrap rounded-md bg-surface-elevated px-2 py-1 text-xs text-foreground opacity-0 shadow ring-1 ring-border transition-opacity",
            "group-hover:opacity-100 group-focus-visible:opacity-100",
            "hidden lg:block xl:hidden"
          )}
          aria-hidden="true"
        >
          {item.label}
        </span>
        {item.comingSoon && (
          <span className="ml-auto hidden rounded bg-surface-elevated px-1.5 py-0.5 text-[10px] text-foreground-muted xl:inline-block">
            Coming soon
          </span>
        )}
      </>
    );

    if (item.comingSoon) {
      return (
        <span
          key={item.href}
          aria-disabled="true"
          className={cn(
            "group relative flex cursor-not-allowed items-center gap-3 rounded-md border-l-2 border-transparent px-3 py-2 text-sm font-medium text-foreground-muted opacity-60",
            "lg:justify-center xl:justify-start"
          )}
          title={`${item.label} — coming soon`}
        >
          {content}
        </span>
      );
    }

    return (
      <Link
        key={item.href}
        href={item.href}
        onClick={onClose}
        aria-current={active ? "page" : undefined}
        title={item.label}
        className={cn(
          "group relative flex items-center gap-3 rounded-md border-l-2 px-3 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50",
          "lg:justify-center xl:justify-start",
          active
            ? "border-primary bg-primary/[0.06] text-primary"
            : "border-transparent text-foreground-secondary hover:bg-surface-elevated hover:text-foreground"
        )}
      >
        {content}
      </Link>
    );
  };

  return (
    <>
      {/* Mobile backdrop */}
      {open && (
        <div
          className="fixed inset-0 z-40 bg-black/60 lg:hidden"
          onClick={onClose}
          aria-hidden="true"
        />
      )}

      {/* Sidebar / drawer */}
      <aside
        ref={drawerRef}
        className={cn(
          "fixed left-0 top-0 z-50 h-full border-r border-border bg-surface transition-transform duration-200",
          "w-60 xl:w-56",
          "max-lg:pt-14",
          // Mobile drawer state: hidden when closed, visible + animated when open
          open ? "translate-x-0 max-lg:block" : "-translate-x-full max-lg:hidden",
          // Desktop: always visible; rail at lg, expanded at xl
          "lg:static lg:w-16 lg:translate-x-0 lg:pt-0 xl:w-56"
        )}
        aria-label="Main navigation"
      >
        <div className="flex h-14 items-center justify-between px-4 lg:hidden">
          <span className="font-semibold text-foreground">Menu</span>
          <IconButton ref={closeButtonRef} icon={X} label="Close navigation" onClick={onClose} />
        </div>

        <nav className="flex h-full flex-col gap-6 px-3 py-4">
          {groups.map((group) => (
            <div key={group.label}>
              <h2
                className={cn(
                  "mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-foreground-muted",
                  "hidden xl:block lg:hidden"
                )}
              >
                {group.label}
              </h2>
              <ul className="space-y-1">
                {group.items.map((item) => (
                  <li key={item.href}>{renderItem(item)}</li>
                ))}
              </ul>
            </div>
          ))}
        </nav>
      </aside>
    </>
  );
}
