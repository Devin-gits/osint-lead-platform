import { Badge } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";

export function CrmConnectorStub() {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">CRM connector</h3>
        <Badge variant="muted">Not configured</Badge>
      </div>
      <p className="text-xs text-foreground-muted">
        Export enriched leads to your CRM (Salesforce, HubSpot, Pipedrive).
        Requires backend integration — not available in pre-production.
      </p>
      <Input
        label="CRM endpoint"
        placeholder="https://api.example.com/leads"
        disabled
      />
    </div>
  );
}
