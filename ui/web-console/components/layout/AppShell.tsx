"use client";

import { useRef, useState } from "react";
import { TopBar } from "./TopBar";
import { ResponsiveSidebar } from "./ResponsiveSidebar";
import { Footer } from "./Footer";

export function AppShell({ children }: { children: React.ReactNode }) {
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const menuButtonRef = useRef<HTMLButtonElement>(null);

  return (
    <div className="flex min-h-screen flex-col bg-background text-foreground">
      <a
        href="#main-content"
        className="skip-link"
      >
        Skip to main content
      </a>

      <ResponsiveSidebar
        open={sidebarOpen}
        onClose={() => setSidebarOpen(false)}
        triggerRef={menuButtonRef}
      />

      <div
        className="flex min-h-screen flex-1 flex-col transition-none lg:ml-16 xl:ml-56"
        id="main-content-wrapper"
      >
        <TopBar
          onMenuClick={() => setSidebarOpen(true)}
          menuButtonRef={menuButtonRef}
        />

        <main
          id="main-content"
          className="flex-1 overflow-auto p-4 md:p-6 lg:p-8"
          tabIndex={-1}
        >
          <div className="mx-auto w-full min-w-0 max-w-[1600px]">{children}</div>
        </main>

        <Footer />
      </div>
    </div>
  );
}
