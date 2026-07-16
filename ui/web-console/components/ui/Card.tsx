import { cn } from "@/lib/utils/cn";
import { HTMLAttributes } from "react";

export type CardProps = HTMLAttributes<HTMLDivElement>;

export function Card({ className, children, ...props }: CardProps) {
  return (
    <div
      className={cn(
        "rounded-lg border border-border bg-surface p-4 shadow-sm",
        className
      )}
      {...props}
    >
      {children}
    </div>
  );
}
