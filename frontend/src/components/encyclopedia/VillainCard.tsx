import Link from "next/link";
import { cn } from "@/components/ui/cn";
import { dossierNumber } from "@/lib/dossier";
import { formatInteger, villainStatusLabels } from "@/lib/format";
import { villainSheetRoute } from "@/lib/routes";
import type { ThreatLevel, Villain, VillainStatus } from "@/types";
import { ThreatMeter } from "./ThreatMeter";
import { VillainMugshot } from "./VillainMugshot";
import { VillainSpecialties } from "./VillainSpecialties";

export const threatTone: Record<ThreatLevel, "sage" | "warning" | "danger"> = {
  faible: "sage",
  modere: "warning",
  eleve: "danger",
  critique: "danger",
};

/** Tampon de statut : rouge s'il court toujours, vert-gris s'il est sous les verrous. */
export function VillainStatusStamp({
  status,
  className,
}: {
  status: VillainStatus;
  className?: string;
}) {
  return (
    <span className={cn("stamp", status === "emprisonne" && "stamp--sage", className)}>
      {villainStatusLabels[status]}
    </span>
  );
}

/**
 * Avis de recherche : photo d'identité, dossier, jauge de menace, spécialités et tampon de statut.
 * Le pseudo est un lien étiré : toute la carte mène à la fiche détaillée.
 */
export function VillainCard({ villain }: { villain: Villain }) {
  return (
    <article className="panel paper paper--wanted flex h-full flex-col p-5 pt-6 focus-within:outline-2 focus-within:outline-offset-4 focus-within:outline-primary">
      <p
        aria-hidden="true"
        className="font-mono text-[0.65rem] tracking-widest text-ink/75 uppercase"
      >
        Dossier {dossierNumber(villain.id, "SV")}
      </p>

      <div className="mt-3 flex items-center gap-4">
        <VillainMugshot alias={villain.alias} size={72} />
        <div className="min-w-0">
          <h3 className="font-display text-lg leading-tight font-bold tracking-wider uppercase">
            <Link
              href={villainSheetRoute(villain.id)}
              className="after:absolute after:inset-0 after:z-10 hover:underline focus-visible:outline-none"
            >
              {villain.alias}
            </Link>
          </h3>
          <ThreatMeter level={villain.threatLevel} className="mt-2" />
        </div>
      </div>

      <p className="mt-4 line-clamp-3 text-sm text-pretty text-ink/85">{villain.description}</p>
      <VillainSpecialties specialties={villain.specialties} className="mt-3" />

      <div className="mt-auto pt-4">
        <div className="flex items-end justify-between gap-3 border-t-2 border-dashed border-ink/30 pt-4">
          <dl>
            <dt className="font-mono text-[0.65rem] tracking-widest text-ink/75 uppercase">
              Incidents signalés
            </dt>
            <dd className="mt-1">
              <span className="digital text-lg">
                {formatInteger(villain.reportedIncidentsCount)}
              </span>
            </dd>
          </dl>
          <VillainStatusStamp status={villain.status} className="mr-1 mb-2" />
        </div>
      </div>
    </article>
  );
}
