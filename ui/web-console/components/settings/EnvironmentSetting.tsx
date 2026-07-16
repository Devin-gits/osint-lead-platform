"use client";

import { Select } from "@/components/ui/Select";
import { useUiStore, type Environment } from "@/lib/store/ui";

const envOptions: { value: Environment; label: string }[] = [
  { value: "sandbox", label: "Sandbox" },
  { value: "production-stub", label: "Production stub" },
];

export function EnvironmentSetting() {
  const { environment, setEnvironment } = useUiStore();

  return (
    <Select
      label="Environment"
      value={environment}
      onChange={(e) => setEnvironment(e.target.value as Environment)}
      options={envOptions}
    />
  );
}
