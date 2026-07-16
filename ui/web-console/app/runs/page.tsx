import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "@/components/ui/PageHeader";
import { Play } from "lucide-react";

export default function RunsPage() {
  return (
    <div className="space-y-6">
      <PageHeader title="Runs" description="Pipeline run history and audit trails." />
      <Card>
        <EmptyState
          icon={Play}
          title="Pipeline runs"
          description="Run timeline and run detail are coming in PR4."
        />
      </Card>
    </div>
  );
}
