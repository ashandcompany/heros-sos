import Link from "next/link";
import type { ComponentProps } from "react";
import { cn } from "./cn";

type Variant = "primary" | "secondary" | "danger" | "ghost" | "console";
type Size = "md" | "lg";

// Boutons brutalistes : cadre épais, ombre dure décalée qui s'écrase au survol et au clic.
const lift =
  "hover:translate-x-px hover:translate-y-px active:translate-x-[3px] active:translate-y-[3px] active:shadow-none";

const variantClasses: Record<Variant, string> = {
  primary: cn(
    "border-2 border-edge bg-primary text-primary-foreground shadow-[4px_4px_0_0_var(--color-edge)] hover:bg-primary-hover hover:shadow-[3px_3px_0_0_var(--color-edge)]",
    lift,
  ),
  secondary: cn(
    "border-2 border-edge bg-surface-raised text-foreground shadow-[4px_4px_0_0_var(--color-edge)] hover:bg-surface hover:shadow-[3px_3px_0_0_var(--color-edge)]",
    lift,
  ),
  // Actions destructrices ou de refus (rejeter, suspendre, supprimer).
  danger: cn(
    "border-2 border-edge bg-brick text-paper shadow-[4px_4px_0_0_var(--color-edge)] hover:bg-[#c9382a] hover:shadow-[3px_3px_0_0_var(--color-edge)]",
    lift,
  ),
  ghost: "text-foreground hover:bg-surface-raised hover:text-accent",
  // Gros bouton d'appel : orange atomique, ombre profonde.
  console: cn(
    "border-[3px] border-edge bg-primary text-primary-foreground shadow-[6px_6px_0_0_var(--color-edge)] hover:bg-primary-hover hover:shadow-[5px_5px_0_0_var(--color-edge)]",
    "hover:translate-x-px hover:translate-y-px active:translate-x-1.5 active:translate-y-1.5 active:shadow-none",
  ),
};

const sizeClasses: Record<Size, string> = {
  md: "min-h-11 px-4 text-sm",
  lg: "min-h-12 px-6 text-base",
};

export const buttonClasses = (variant: Variant = "primary", size: Size = "md") =>
  cn(
    "inline-flex items-center justify-center gap-2 font-display font-bold tracking-[0.1em] uppercase",
    "transition-none",
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
