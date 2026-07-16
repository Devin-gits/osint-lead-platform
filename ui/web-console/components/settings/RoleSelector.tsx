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
    <Select
      label="Active role"
      value={role}
      onChange={(e) => setRole(e.target.value as Role)}
      options={roleOptions}
    />
  );
}
