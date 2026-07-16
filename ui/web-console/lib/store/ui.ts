import { create } from "zustand";

export type Role = "sales" | "admin" | "risk";
export type Environment = "sandbox" | "production-stub";

interface UiState {
  role: Role;
  environment: Environment;
  setRole: (role: Role) => void;
  setEnvironment: (environment: Environment) => void;
}

export const useUiStore = create<UiState>((set) => ({
  role: "sales",
  environment: "sandbox",
  setRole: (role) => set({ role }),
  setEnvironment: (environment) => set({ environment }),
}));
