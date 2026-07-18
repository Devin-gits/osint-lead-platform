import { Badge } from "@/components/ui/Badge";
import { Input } from "@/components/ui/Input";

export function SsoOidStub() {
  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="text-sm font-semibold text-foreground">SSO / OIDC provider</h3>
        <Badge variant="outline">Coming soon</Badge>
      </div>
      <p className="text-xs text-foreground-muted">
        Single sign-on via OpenID Connect. Will enforce role-based access once
        authentication is integrated. Currently all access is unauthenticated.
      </p>
      <Input
        label="Issuer URL"
        placeholder="https://auth.example.com/.well-known/openid-configuration"
        disabled
      />
    </div>
  );
}
