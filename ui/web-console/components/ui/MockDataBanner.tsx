import { AlertTriangle } from "lucide-react";

export function MockDataBanner() {
  return (
    <div
      className="flex items-center justify-center gap-2 border-b border-primary/20 bg-primary/10 px-4 py-2 text-xs font-medium text-primary"
      role="status"
    >
      <AlertTriangle className="h-4 w-4" aria-hidden="true" />
      <span>Mock data — backend not wired yet</span>
    </div>
  );
}
