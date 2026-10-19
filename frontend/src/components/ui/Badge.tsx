import type { ReactNode } from "react";
import { cn } from "./cn";

type Tone = "neutral" | "paper" | "accent" | "success" | "warning" | "danger" | "sage";

/**
 * Teintes des statuts d'incident et des priorités :
 * paper = signalé, warning/accent = en cours (ambre), success = réussi (vert phosphore),
 * danger = échoué / priorité haute (rouge brique), sage = priorité basse.
 */
const toneClasses: Record<Tone, string> = {
  neutral: "border-border bg-surface-raised text-foreground",
  paper: "border-paper/50 bg-paper/10 text-paper",
  accent: "border-accent/50 bg-accent/10 text-accent",
  success: "border-success/50 bg-success/10 text-success",
  warning: "border-rating/50 bg-rating/10 text-rating",
  danger: "border-brick bg-brick text-paper",
  sage: "border-sage bg-sage text-paper",
};

export function Badge({ tone = "neutral", children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-sm border px-2 py-0.5 font-mono text-xs font-medium tracking-wider uppercase",
        toneClasses[tone],
      )}
    >
      {children}
    </span>
  );
}
