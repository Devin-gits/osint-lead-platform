"use client";

import { useState } from "react";
import { Button } from "@/components/ui/Button";
import { IconButton } from "@/components/ui/IconButton";
import { Input } from "@/components/ui/Input";
import { Select } from "@/components/ui/Select";
import { Textarea } from "@/components/ui/Textarea";
import { Card } from "@/components/ui/Card";
import { Badge } from "@/components/ui/Badge";
import {
  Table,
  TableHead,
  TableBody,
  TableRow,
  TableHeader,
  TableCell,
} from "@/components/ui/Table";
import { Tabs } from "@/components/ui/Tabs";
import { Dialog } from "@/components/ui/Dialog";
import { Toast } from "@/components/ui/Toast";
import { Tooltip } from "@/components/ui/Tooltip";
import { Skeleton } from "@/components/ui/Skeleton";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "@/components/ui/PageHeader";
import { StatusChip } from "@/components/ui/StatusChip";
import { PipelineStepper } from "@/components/ui/PipelineStepper";
import { AuditLogPanel } from "@/components/ui/AuditLogPanel";
import { Info, Trash2 } from "lucide-react";

export default function StyleGuidePage() {
  const [dialogOpen, setDialogOpen] = useState(false);

  return (
    <div className="space-y-8 pb-12">
      <PageHeader title="Style guide" description="Design-system smoke test for PR1." />

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Buttons
        </h2>
        <div className="flex flex-wrap gap-2">
          <Button>Primary</Button>
          <Button variant="secondary">Secondary</Button>
          <Button variant="ghost">Ghost</Button>
          <Button variant="danger">Danger</Button>
          <IconButton icon={Trash2} label="Delete" />
          <Tooltip content="More info">
            <Info className="h-5 w-5 text-foreground-muted" />
          </Tooltip>
        </div>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Form controls
        </h2>
        <div className="grid gap-4 sm:grid-cols-2">
          <Input label="Email" placeholder="lead@example.com" />
          <Select
            label="Stage"
            options={[
              { value: "raw", label: "Raw" },
              { value: "enriched", label: "Enriched" },
              { value: "validated", label: "Validated" },
            ]}
          />
          <Textarea label="Notes" placeholder="Enter notes..." className="sm:col-span-2" />
        </div>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Badges & status
        </h2>
        <div className="flex flex-wrap gap-2">
          <Badge>primary</Badge>
          <Badge variant="secondary">secondary</Badge>
          <Badge variant="success">success</Badge>
          <Badge variant="warning">warning</Badge>
          <Badge variant="danger">danger</Badge>
          <Badge variant="muted">muted</Badge>
          <StatusChip status="ok" />
          <StatusChip status="unknown" />
          <StatusChip status="skipped" />
          <StatusChip status="pending" />
          <StatusChip status="not_run" />
        </div>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Table
        </h2>
        <Card>
          <Table>
            <TableHead>
              <TableRow>
                <TableHeader>Name</TableHeader>
                <TableHeader>Stage</TableHeader>
                <TableHeader>Risk</TableHeader>
              </TableRow>
            </TableHead>
            <TableBody>
              <TableRow>
                <TableCell>Acme Corp</TableCell>
                <TableCell>enriched</TableCell>
                <TableCell>low</TableCell>
              </TableRow>
            </TableBody>
          </Table>
        </Card>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Tabs
        </h2>
        <Card>
          <Tabs
            tabs={[
              { id: "email", label: "Email", content: <p className="text-sm text-foreground-secondary">Email validation results go here.</p> },
              { id: "phone", label: "Phone", content: <p className="text-sm text-foreground-secondary">Phone validation results go here.</p> },
            ]}
          />
        </Card>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Dialog
        </h2>
        <Button onClick={() => setDialogOpen(true)}>Open dialog</Button>
        <Dialog open={dialogOpen} onClose={() => setDialogOpen(false)} title="Example dialog">
          <p className="text-sm text-foreground-secondary">This is a modal dialog shell.</p>
          <div className="mt-4 flex justify-end gap-2">
            <Button variant="ghost" onClick={() => setDialogOpen(false)}>
              Close
            </Button>
          </div>
        </Dialog>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Toast & skeleton
        </h2>
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
          <Toast message="This is a toast message" />
          <Skeleton className="h-8 w-48" />
        </div>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Pipeline stepper
        </h2>
        <Card>
          <PipelineStepper current="validated" />
        </Card>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Audit log panel
        </h2>
        <Card>
          <AuditLogPanel
            events={[
              {
                id: "evt-1",
                module: "email-validate",
                tool: "AfterShip/email-verifier@v1.4.1",
                checked_at: "2026-07-13T13:45:46Z",
                status: "ok",
                legal_basis: "GDPR Art.6(1)(f) legitimate-interest",
                raw_stderr_json: '{"tool":"AfterShip/email-verifier@v1.4.1","status":"ok"}',
              },
            ]}
          />
        </Card>
      </section>

      <section className="space-y-2">
        <h2 className="text-sm font-semibold uppercase tracking-wider text-foreground-muted">
          Empty state
        </h2>
        <Card>
          <EmptyState title="Nothing to see" description="This is an empty state placeholder." />
        </Card>
      </section>
    </div>
  );
}
