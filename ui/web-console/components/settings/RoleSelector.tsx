"use client";

import { Select } from "@/components/ui/Select";
import { useUiStore, type Role } from "@/lib/store/ui";

const roleOptions: { value: Role; label: string }[] = [
  { value: "sales", label: "Sales / Ops" },
  { value: "admin", label: "Admin" },
  { value: "risk", label: "Risk / Compliance" },
];

export function RoleSelector() {
  const { role, setRole } = useUiStore();

  return (
    <div className="space-y-3">
      <h3 className="text-sm font-semibold text-foreground">Active role</h3>
      <p className="text-xs text-foreground-muted">
        UI preview only — not authorization. Changes the view perspective but
        does not enforce access control until SSO integration is complete.
      </p>
      <Select
        label="Role"
        value={role}
        onChange={(e) => setRole(e.target.value as Role)}
        options={roleOptions}
      />
    </div>
  );
}
