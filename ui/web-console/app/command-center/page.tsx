import type { ReactNode } from "react";
import Link from "next/link";
import { cn } from "@/lib/utils/cn";
import { Card } from "@/components/ui/Card";
import { APIHealthIndicator } from "@/components/ui/APIHealthIndicator";
import {
  Users,
  Play,
  Box,
  Shield,
  FileText,
  ArrowRight,
  CheckCircle2,
} from "lucide-react";

export const metadata = {
  title: "Command Center | OSINT Lead Console",
};

const shortcuts = [
  { href: "/leads", label: "Review leads", description: "Inspect and run checks on permissioned leads.", icon: Users },
  { href: "/runs", label: "View runs", description: "See batch and single-lead pipeline execution history.", icon: Play },
  { href: "/modules", label: "Explore modules", description: "Browse available checks, docs, and input requirements.", icon: Box },
  { href: "/compliance", label: "Review compliance", description: "Read hard rules, risk table, and pre-run checklist.", icon: Shield },
];

const workflow = [
  "Create a lead with a permission reference",
  "Select approved checks",
  "Review validation results",
  "Inspect activity and audit evidence",
];

function ButtonLink({
  href,
  variant,
  children,
  className,
}: {
  href: string;
  variant: "primary" | "ghost";
  children: ReactNode;
  className?: string;
}) {
  return (
    <Link
      href={href}
      className={cn(
        "inline-flex items-center justify-center rounded-md px-4 py-2 text-sm font-medium transition-colors focus:outline-none focus:ring-2",
        variant === "primary" &&
          "bg-primary text-background hover:brightness-110 focus:ring-primary/50",
        variant === "ghost" &&
          "bg-transparent text-foreground hover:bg-surface-elevated focus:ring-primary/50",
        className
      )}
    >
      {children}
    </Link>
  );
}

export default function CommandCenterPage() {
  return (
    <div className="space-y-8">
      <section className="space-y-4">
        <h1 className="text-2xl font-semibold tracking-tight text-foreground md:text-3xl">
          Command Center
        </h1>
        <p className="max-w-3xl text-foreground-secondary">
          Validate and enrich permissioned leads, then review module results and
          compliance evidence.
        </p>
        <div className="flex flex-wrap gap-3">
          <ButtonLink href="/leads/create" variant="primary">
            Create lead
            <ArrowRight className="ml-1.5 h-4 w-4" aria-hidden="true" />
          </ButtonLink>
          <ButtonLink href="/leads" variant="ghost">
            Review leads
          </ButtonLink>
        </div>
      </section>

      <section className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {shortcuts.map(({ href, label, description, icon: Icon }) => (
          <Card
            key={href}
            className="flex h-full flex-col justify-between p-4 transition-colors hover:border-primary/30"
          >
            <div>
              <div className="mb-3 flex items-center gap-2 text-foreground">
                <Icon className="h-5 w-5 text-primary" aria-hidden="true" />
                <h2 className="font-medium">{label}</h2>
              </div>
              <p className="text-sm text-foreground-secondary">{description}</p>
            </div>
            <div className="mt-4">
              <Link
                href={href}
                className="inline-flex items-center text-sm font-medium text-primary hover:underline focus:outline-none focus:ring-2 focus:ring-primary/50"
              >
                Go to {label.toLowerCase()}
                <ArrowRight className="ml-1 h-3.5 w-3.5" aria-hidden="true" />
              </Link>
            </div>
          </Card>
        ))}
      </section>

      <section className="grid gap-6 lg:grid-cols-2">
        <Card className="p-5">
          <div className="mb-4 flex items-center gap-2">
            <FileText className="h-5 w-5 text-primary" aria-hidden="true" />
            <h2 className="font-semibold text-foreground">Workflow</h2>
          </div>
          <ol className="space-y-3">
            {workflow.map((step, index) => (
              <li key={step} className="flex items-start gap-3 text-sm text-foreground-secondary">
                <span className="flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-surface-elevated text-xs font-medium text-foreground">
                  {index + 1}
                </span>
                {step}
              </li>
            ))}
          </ol>
        </Card>

        <Card className="flex flex-col justify-between p-5">
          <div>
            <div className="mb-4 flex items-center gap-2">
              <CheckCircle2 className="h-5 w-5 text-primary" aria-hidden="true" />
              <h2 className="font-semibold text-foreground">API reachability</h2>
            </div>
            <p className="mb-4 text-sm text-foreground-secondary">
              This indicator only confirms that the control-plane API responds
              to a lightweight request. It does not claim database, runner, or
              module health.
            </p>
          </div>
          <APIHealthIndicator />
        </Card>
      </section>
    </div>
  );
}
