import Link from "next/link";
import { Badge } from "@/components/ui/Badge";
import { formatInteger, threatLevelLabels } from "@/lib/format";
import { villainSheetRoute } from "@/lib/routes";
import type { ThreatLevel, Villain } from "@/types";

export const threatTone: Record<ThreatLevel, "sage" | "warning" | "danger"> = {
  faible: "sage",
  modere: "warning",
  eleve: "danger",
  critique: "danger",
};

export function VillainCard({ villain }: { villain: Villain }) {
  return (
    <article className="relative flex h-full flex-col border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-primary hover:border-muted">
      <div className="flex items-start justify-between gap-3">
        <h3 className="font-display text-lg font-semibold">
          {/* Lien étiré : toute la carte mène à la fiche détaillée. */}
          <Link
            href={villainSheetRoute(villain.id)}
            className="after:absolute after:inset-0 after:rounded-2xl focus-visible:outline-none"
          >
            {villain.alias}
          </Link>
        </h3>
        <Badge tone={threatTone[villain.threatLevel]}>
          <span className="sr-only">Menace : </span>
          {threatLevelLabels[villain.threatLevel]}
        </Badge>
      </div>
      <p className="mt-2 text-sm text-muted text-pretty">{villain.description}</p>
      {villain.status === "emprisonne" && (
        <p className="mt-3">
          <Badge tone="sage">Emprisonné</Badge>
        </p>
      )}
      <p className="mt-auto pt-4 text-sm">
        <span className="font-mono font-semibold">
          {formatInteger(villain.reportedIncidentsCount)}
        </span>{" "}
        <span className="text-muted">incidents signalés</span>
      </p>
    </article>
  );
}
