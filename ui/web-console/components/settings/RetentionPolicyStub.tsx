import { Badge } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";

export function RetentionPolicyStub() {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">Data retention policy</h3>
        <Badge variant="outline">Coming soon</Badge>
      </div>
      <p className="text-xs text-foreground-muted">
        Configure how long enrichment results are retained before automatic
        deletion or CRM export. Will be enforced by the control-plane scheduler.
      </p>
      <Input
        label="Retention window"
        placeholder="30d (defined by future backend)"
        disabled
      />
    </div>
  );
}
