import { cn } from "@/lib/utils/cn";

export type PipelineStep = "raw" | "enriched" | "validated" | "crm_ready";

export interface PipelineStepperProps {
  current: PipelineStep;
  className?: string;
}

const steps: PipelineStep[] = ["raw", "enriched", "validated", "crm_ready"];

export function PipelineStepper({ current, className }: PipelineStepperProps) {
  const currentIndex = steps.indexOf(current);

  return (
    <div className={cn("flex items-center gap-2", className)}>
      {steps.map((step, index) => {
        const isActive = index <= currentIndex;
        return (
          <div key={step} className="flex items-center gap-2">
            <div
              className={cn(
                "flex h-6 w-6 items-center justify-center rounded-full text-xs font-semibold",
                isActive
                  ? "bg-primary text-background"
                  : "bg-surface-elevated text-foreground-muted border border-border"
              )}
            >
              {index + 1}
            </div>
            <span
              className={cn(
                "text-xs font-medium capitalize",
                isActive ? "text-primary" : "text-foreground-muted"
              )}
            >
              {step.replace("_", " ")}
            </span>
            {index < steps.length - 1 && (
              <div
                className={cn(
                  "mx-1 h-px w-6",
                  index < currentIndex ? "bg-primary" : "bg-border"
                )}
              />
            )}
          </div>
        );
      })}
    </div>
  );
}
