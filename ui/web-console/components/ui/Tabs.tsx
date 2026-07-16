"use client";

import { cn } from "@/lib/utils/cn";
import { useState } from "react";

export interface Tab {
  id: string;
  label: string;
  content: React.ReactNode;
}

export interface TabsProps {
  tabs: Tab[];
  defaultTab?: string;
}

export function Tabs({ tabs, defaultTab }: TabsProps) {
  const [active, setActive] = useState(defaultTab || tabs[0]?.id);

  return (
    <div className="space-y-4">
      <div className="flex border-b border-border">
        {tabs.map((tab) => (
          <button
            key={tab.id}
            onClick={() => setActive(tab.id)}
            className={cn(
              "px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2 focus:ring-primary/50",
              active === tab.id
                ? "border-b-2 border-primary text-primary"
                : "text-foreground-muted hover:text-foreground"
            )}
            aria-selected={active === tab.id}
            role="tab"
          >
            {tab.label}
          </button>
        ))}
      </div>
      <div role="tabpanel" className="min-h-[4rem]">
        {tabs.find((t) => t.id === active)?.content}
      </div>
    </div>
  );
}
