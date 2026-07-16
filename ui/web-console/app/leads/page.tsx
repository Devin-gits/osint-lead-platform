import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "@/components/ui/PageHeader";
import { Users } from "lucide-react";

export default function LeadsPage() {
  return (
    <div className="space-y-6">
      <PageHeader title="Leads" description="Manage and inspect enriched leads." />
      <Card>
        <EmptyState
          icon={Users}
          title="Leads list"
          description="Filters, table, and lead detail are coming in PR2 with the mock API."
        />
      </Card>
    </div>
  );
}
