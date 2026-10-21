import { cn } from "@/components/ui/cn";
import { threatLevelLabels } from "@/lib/format";
import type { ThreatLevel } from "@/types";

const levels: ThreatLevel[] = ["faible", "modere", "eleve", "critique"];

/** Mêmes couleurs que les priorités des repères de la carte. */
const fillClasses: Record<ThreatLevel, string> = {
  faible: "bg-sage",
  modere: "bg-mustard",
  eleve: "bg-primary",
  critique: "bg-brick",
};

/**
 * Jauge de menace à quatre crans, à poser sur une fiche papier.
 * Le niveau se lit au nombre de crans et au libellé, pas seulement à la couleur.
 */
export function ThreatMeter({ level, className }: { level: ThreatLevel; className?: string }) {
  const rank = levels.indexOf(level);
  return (
    <span className={cn("inline-flex items-center gap-2", className)}>
      <span aria-hidden="true" className="flex gap-0.5">
        {levels.map((step, index) => (
          <span
            key={step}
            className={cn("h-3 w-4 border-2 border-ink", index <= rank && fillClasses[level])}
          />
        ))}
      </span>
      <span className="font-mono text-xs font-semibold tracking-wider uppercase">
        <span className="sr-only">Menace : </span>
        {threatLevelLabels[level]}
      </span>
    </span>
  );
}
