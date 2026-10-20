import type { ReactNode } from "react";
import { dossierNumber } from "@/lib/dossier";
import { formatDate, formatLocation } from "@/lib/format";
import type { ResolvedIncident } from "@/types";

/** Ligne de formulaire : intitulé imprimé, valeur tapée à la machine sur une ligne pointillée. */
function Field({ label, children }: { label: string; children: ReactNode }) {
  return (
    <div className="grid grid-cols-[4.5rem_minmax(0,1fr)] items-baseline gap-2">
      <dt className="font-mono text-[0.65rem] tracking-widest text-ink/70 uppercase">{label}</dt>
      <dd className="field-line truncate pb-0.5 font-stamp text-[0.95rem] leading-snug">
        {children}
      </dd>
    </div>
  );
}

/** Perforation de classeur : on voit le fond à travers la feuille. */
function PunchHole({ className }: { className: string }) {
  return (
    <span
      aria-hidden="true"
      className={`absolute left-2.5 size-3 rounded-full bg-background shadow-[inset_1px_1px_2px_rgb(0_0_0/0.6)] ${className}`}
    />
  );
}

/**
 * Fiche d'archive cartonnée : onglet de chemise, feuille perforée à marge rouge, tampon
 * « Résolu », champs tapés à la machine et visa de l'agent.
 * L'onglet est réservé dans le padding (et non une marge) pour que la carte ne dépasse pas de
 * sa diapositive : sinon le carrousel défile verticalement.
 */
export function IncidentCard({ incident }: { incident: ResolvedIncident }) {
  const number = dossierNumber(incident.id);

  return (
    <article className="relative flex h-full flex-col pt-7">
      {/* Onglet de la chemise, soudé à la feuille (il recouvre sa bordure supérieure). */}
      <div
        aria-hidden="true"
        className="paper absolute top-0 left-5 z-10 h-[calc(1.75rem+2px)] border-2 border-b-0 border-edge px-3 pt-1 font-mono text-xs font-medium tracking-widest uppercase"
      >
        Dossier {number}
      </div>

      <div className="paper relative flex flex-1 flex-col border-2 border-edge py-4 pr-4 pl-10 shadow-[4px_4px_0_0_var(--color-shadow-deep)] sm:pr-5">
        <p className="sr-only">Dossier {number}, statut : résolu.</p>

        {/* Marge rouge et perforations, comme une feuille de classeur. */}
        <span aria-hidden="true" className="absolute inset-y-0 left-7 border-l border-brick/45" />
        <PunchHole className="top-[22%]" />
        <PunchHole className="bottom-[22%]" />

        <p
          aria-hidden="true"
          className="truncate pr-24 font-mono text-[0.6rem] tracking-widest text-ink/70 uppercase"
        >
          <span>Heros-SOS · Archives</span>
        </p>
        <div aria-hidden="true" className="mt-1.5 border-t-2 border-double border-ink/40" />

        {/* Tampon apposé à droite de l'en-tête : il ne recouvre aucun champ. */}
        <span
          aria-hidden="true"
          className="stamp absolute top-3 right-3 text-lg mix-blend-multiply"
        >
          Résolu
        </span>

        <h3 className="mt-3 line-clamp-2 min-h-[2.75rem] font-display text-base leading-snug font-bold tracking-wider text-balance uppercase">
          {incident.title}
        </h3>

        <dl className="mt-3 space-y-2 text-sm">
          <Field label="Nature">{incident.type.label}</Field>
          <Field label="Lieu">{formatLocation(incident.location)}</Field>
          <Field label="Agent">{incident.hero.alias}</Field>
          <Field label="Suspect">{incident.villain?.alias ?? "Non identifié"}</Field>
        </dl>

        {/* Pied de fiche : date de classement et visa de l'agent. */}
        <div className="mt-auto flex items-end justify-between gap-3 pt-5">
          <p className="font-mono text-[0.65rem] tracking-widest text-ink/70 uppercase">
            Classé le
            <time
              dateTime={incident.resolvedAt}
              className="mt-0.5 block font-stamp text-sm tracking-normal text-ink normal-case"
            >
              {formatDate(incident.resolvedAt)}
            </time>
          </p>
          <p className="min-w-0 text-right font-mono text-[0.65rem] tracking-widest text-ink/70 uppercase">
            Visa de l&apos;agent
            <span className="mt-0.5 block border-b border-ink/50 px-1 font-stamp text-base tracking-normal text-ink normal-case italic">
              {incident.hero.alias}
            </span>
          </p>
        </div>
      </div>
    </article>
  );
}
