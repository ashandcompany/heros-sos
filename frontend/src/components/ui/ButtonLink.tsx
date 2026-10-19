import Link from "next/link";
import type { ComponentProps } from "react";
import { cn } from "./cn";

type Variant = "primary" | "secondary" | "ghost";
type Size = "md" | "lg";

const variantClasses: Record<Variant, string> = {
  primary: "bg-primary text-primary-foreground hover:bg-primary-hover",
  secondary: "border border-border text-foreground hover:border-muted hover:bg-surface-raised",
  ghost: "text-foreground hover:bg-surface-raised",
};

const sizeClasses: Record<Size, string> = {
  md: "min-h-11 px-4 text-sm",
  lg: "min-h-12 px-6 text-base",
};

export const buttonClasses = (variant: Variant = "primary", size: Size = "md") =>
  cn(
    "inline-flex items-center justify-center gap-2 rounded-full font-semibold transition-colors motion-reduce:transition-none",
    variantClasses[variant],
    sizeClasses[size],
  );

type ButtonLinkProps = ComponentProps<typeof Link> & {
  variant?: Variant;
  size?: Size;
};

export function ButtonLink({ variant, size, className, ...props }: ButtonLinkProps) {
  return <Link className={cn(buttonClasses(variant, size), className)} {...props} />;
}
