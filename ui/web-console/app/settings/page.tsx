import { Card } from "@/components/ui/Card";
import { PageHeader } from "@/components/ui/PageHeader";
import { EnvironmentSetting } from "@/components/settings/EnvironmentSetting";
import { RoleSelector } from "@/components/settings/RoleSelector";
import { CrmConnectorStub } from "@/components/settings/CrmConnectorStub";
import { SsoOidStub } from "@/components/settings/SsoOidStub";
import { ApiKeysVaultStub } from "@/components/settings/ApiKeysVaultStub";
import { RetentionPolicyStub } from "@/components/settings/RetentionPolicyStub";

export default function SettingsPage() {
  return (
    <div className="space-y-6">
      <PageHeader title="Settings" description="Environment, role, and connector stubs." />
      <div className="grid gap-4 sm:grid-cols-2">
        <Card>
          <div className="space-y-4">
            <EnvironmentSetting />
            <RoleSelector />
          </div>
        </Card>
        <Card>
          <div className="space-y-4">
            <CrmConnectorStub />
            <SsoOidStub />
          </div>
        </Card>
        <Card>
          <ApiKeysVaultStub />
        </Card>
        <Card>
          <RetentionPolicyStub />
        </Card>
      </div>
    </div>
  );
}
