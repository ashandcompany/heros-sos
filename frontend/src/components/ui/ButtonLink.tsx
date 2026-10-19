import Link from "next/link";
import type { ComponentProps } from "react";
import { cn } from "./cn";

type Variant = "primary" | "secondary" | "ghost" | "console";
type Size = "md" | "lg";

const variantClasses: Record<Variant, string> = {
  // Touches de console : reflet en haut, tranche en bas qui s'écrase au clic.
  primary:
    "border border-[#8f3a12] bg-primary text-primary-foreground shadow-[inset_0_1px_0_rgb(255_255_255/0.3),inset_0_-2px_0_rgb(0_0_0/0.2),0_3px_0_#7a2f0f,0_4px_0_#000] hover:bg-primary-hover active:translate-y-0.5 active:shadow-[inset_0_1px_0_rgb(255_255_255/0.2),0_1px_0_#7a2f0f,0_2px_0_#000]",
  secondary:
    "border border-border bg-surface-raised text-foreground shadow-[inset_0_1px_0_rgb(255_255_255/0.12),inset_0_-2px_0_rgb(0_0_0/0.25),0_3px_0_#0d0906] hover:bg-surface hover:border-muted active:translate-y-0.5 active:shadow-[0_1px_0_#0d0906]",
  ghost: "text-foreground hover:bg-surface-raised hover:text-accent",
  // Gros bouton de console industrielle.
  console:
    "border-2 border-[#8f3a12] bg-primary text-primary-foreground shadow-[inset_0_2px_0_rgb(255_255_255/0.3),inset_0_-4px_0_rgb(0_0_0/0.2),0_7px_0_#7a2f0f,0_9px_0_#000] hover:bg-primary-hover active:translate-y-1.5 active:shadow-[inset_0_2px_0_rgb(255_255_255/0.2),0_1px_0_#7a2f0f,0_3px_0_#000]",
};

const sizeClasses: Record<Size, string> = {
  md: "min-h-11 px-4 text-sm",
  lg: "min-h-12 px-6 text-base",
};

export const buttonClasses = (variant: Variant = "primary", size: Size = "md") =>
  cn(
    "inline-flex items-center justify-center gap-2 rounded-md font-display font-semibold tracking-[0.08em] uppercase",
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
