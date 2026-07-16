import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "@/components/ui/PageHeader";
import { Box } from "lucide-react";

export default function ModulesPage() {
  return (
    <div className="space-y-6">
      <PageHeader title="Modules" description="Module status, configuration, and documentation." />
      <Card>
        <EmptyState
          icon={Box}
          title="Module dashboard"
          description="Module grid and detail views are coming in PR3."
        />
      </Card>
    </div>
  );
}
