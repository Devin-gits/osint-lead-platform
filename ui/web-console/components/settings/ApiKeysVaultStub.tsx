import { Textarea } from "@/components/ui/Textarea";

export function ApiKeysVaultStub() {
  return (
    <Textarea
      label="API keys vault"
      placeholder="Vault not configured (stub) — do not store real secrets here."
      disabled
    />
  );
}
