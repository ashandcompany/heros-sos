import { Badge } from "@/components/ui/Badge";
import { formatInteger, threatLevelLabels } from "@/lib/format";
import type { ThreatLevel, Villain } from "@/types";

const threatTone: Record<ThreatLevel, "neutral" | "warning" | "danger"> = {
  faible: "neutral",
  modere: "warning",
  eleve: "danger",
  critique: "danger",
};

export function VillainCard({ villain }: { villain: Villain }) {
  return (
    <article className="flex h-full flex-col rounded-2xl border border-border bg-surface p-5">
      <div className="flex items-start justify-between gap-3">
        <h3 className="font-display text-lg font-semibold">{villain.alias}</h3>
        <Badge tone={threatTone[villain.threatLevel]}>
          <span className="sr-only">Menace : </span>
          {threatLevelLabels[villain.threatLevel]}
        </Badge>
      </div>
      <p className="mt-2 text-sm text-muted text-pretty">{villain.description}</p>
      <p className="mt-auto pt-4 text-sm">
        <span className="font-semibold">{formatInteger(villain.reportedIncidentsCount)}</span>{" "}
        <span className="text-muted">incidents signalés</span>
      </p>
    </article>
  );
}
