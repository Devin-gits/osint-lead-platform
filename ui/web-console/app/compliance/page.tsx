import { Card } from "@/components/ui/Card";
import { EmptyState } from "@/components/ui/EmptyState";
import { PageHeader } from "@/components/ui/PageHeader";
import { Shield } from "lucide-react";

export default function CompliancePage() {
  return (
    <div className="space-y-6">
      <PageHeader title="Compliance" description="Hard rules, risk table, and pre-run checklist." />
      <Card>
        <EmptyState
          icon={Shield}
          title="Compliance center"
          description="Hard rules, risk table, and interactive checklist are coming in PR4."
        />
      </Card>
    </div>
  );
}
