import type { ReactNode } from "react";
import { cn } from "./cn";

type Tone = "neutral" | "paper" | "accent" | "success" | "warning" | "danger" | "sage";

/**
 * Teintes des statuts d'incident et des priorités :
 * paper = signalé, warning/accent = en cours (ambre), success = réussi (vert phosphore),
 * danger = échoué / priorité haute (rouge brique), sage = priorité basse.
 */
const toneClasses: Record<Tone, string> = {
  neutral: "border-edge bg-surface-raised text-foreground",
  // Texte clair sur fond sombre, foncé sur fond clair (foreground = papier en thème sombre).
  paper: "border-foreground bg-transparent text-foreground",
  accent: "border-accent bg-transparent text-accent",
  success: "border-success bg-transparent text-success",
  warning: "border-rating bg-transparent text-rating",
  danger: "border-brick bg-brick text-paper",
  sage: "border-sage bg-sage text-paper",
};

export function Badge({ tone = "neutral", children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 border-2 px-2 py-0.5 font-mono text-xs font-semibold tracking-wider uppercase",
        toneClasses[tone],
      )}
    >
      {children}
    </span>
  );
}
