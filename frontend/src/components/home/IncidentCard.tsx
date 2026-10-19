import type { ReactNode } from "react";
import { formatDate, formatLocation, incidentTypeLabels } from "@/lib/format";
import { dossierNumber } from "@/lib/dossier";
import type { ResolvedIncident } from "@/types";

function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="flex flex-col gap-0.5 sm:flex-row sm:items-baseline sm:gap-3">
      <dt className="shrink-0 font-mono text-[0.7rem] tracking-widest text-background/75 uppercase sm:w-28">
        {label}
      </dt>
      <dd className="field-line min-w-0 flex-1 pb-0.5 font-semibold">{children}</dd>
    </div>
  );
}

/** Fiche de dossier cartonné : onglet, numéro, champs typés, tampon « RÉSOLU ». */
export function IncidentCard({ incident }: { incident: ResolvedIncident }) {
  return (
    <article className="relative mt-7 flex h-full flex-col">
      {/* Onglet du dossier */}
      <div
        aria-hidden="true"
        className="paper absolute -top-7 left-4 border-2 border-b-0 border-[#0d0906] px-3 py-1 font-mono text-xs font-medium tracking-widest uppercase"
      >
        Dossier {dossierNumber(incident.id)}
      </div>

      <div className="paper relative flex h-full flex-col border-2 border-[#0d0906] p-4 shadow-[4px_4px_0_0_#000] sm:p-5">
        <p className="sr-only">Dossier {dossierNumber(incident.id)}, statut : résolu.</p>

        <h3 className="pr-24 font-display text-base leading-snug font-bold tracking-wider text-balance uppercase">
          {incident.title}
        </h3>
        <div aria-hidden="true" className="rule-thick mt-3 mb-4" />

        <dl className="space-y-2.5 text-sm">
          <Field label="Type">{incidentTypeLabels[incident.type]}</Field>
          <Field label="Lieu">{formatLocation(incident.location)}</Field>
          <Field label="Date">
            <time dateTime={incident.resolvedAt} className="font-mono">
              {formatDate(incident.resolvedAt)}
            </time>
          </Field>
          <Field label="Agent intervenant">{incident.hero.alias}</Field>
          <Field label="Sujet appréhendé">{incident.villain?.alias ?? "Non identifié"}</Field>
        </dl>

        <span aria-hidden="true" className="stamp absolute top-3 right-3 text-lg">
          Résolu
        </span>
      </div>
    </article>
  );
}
