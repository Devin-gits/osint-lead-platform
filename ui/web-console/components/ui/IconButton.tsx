"use client";

import { cn } from "@/lib/utils/cn";
import { ButtonHTMLAttributes, forwardRef } from "react";
import { LucideIcon } from "lucide-react";

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  icon: LucideIcon;
  label: string;
}

export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(
  ({ icon: Icon, label, className, ...props }, ref) => {
    return (
      <button
        ref={ref}
        aria-label={label}
        title={label}
        className={cn(
          "inline-flex items-center justify-center rounded-md p-2 text-foreground hover:bg-surface-elevated focus:outline-none focus:ring-2 focus:ring-primary/50 disabled:opacity-50",
          className
        )}
        {...props}
      >
        <Icon className="h-5 w-5" />
      </button>
    );
  }
);
IconButton.displayName = "IconButton";
