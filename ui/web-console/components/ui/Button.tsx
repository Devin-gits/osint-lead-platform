"use client";

import { cn } from "@/lib/utils/cn";
import { ButtonHTMLAttributes, forwardRef } from "react";

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  size?: "sm" | "md" | "lg";
}

export const Button = forwardRef<HTMLButtonElement, ButtonProps>(
  ({ variant = "primary", size = "md", className, ...props }, ref) => {
    const variants = {
      primary:
        "bg-primary text-background hover:brightness-110 focus:ring-2 focus:ring-primary/50",
      secondary:
        "bg-secondary text-white hover:brightness-110 focus:ring-2 focus:ring-secondary/50",
      ghost:
        "bg-transparent text-foreground hover:bg-surface-elevated focus:ring-2 focus:ring-primary/50",
      danger:
        "bg-danger text-white hover:brightness-110 focus:ring-2 focus:ring-danger/50",
    };

    const sizes = {
      sm: "px-2.5 py-1 text-xs",
      md: "px-4 py-2 text-sm",
      lg: "px-6 py-2.5 text-base",
    };

    return (
      <button
        ref={ref}
        className={cn(
          "inline-flex items-center justify-center rounded-md font-medium transition-colors focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed",
          variants[variant],
          sizes[size],
          className
        )}
        {...props}
      />
    );
  }
);
Button.displayName = "Button";
