"use client";

import { useActionState, useTransition } from "react";
import { setLocationSharingAction } from "@/app/actions/hero";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { formatDate } from "@/lib/format";
import { idleFormState } from "@/lib/validation/form-state";
import { type HeroLocationStatus, useHeroLocation } from "./HeroLocation";

const statusMessages: Record<HeroLocationStatus, string> = {
  off: "Votre position n'est pas envoyée pour le moment.",
  locating: "Localisation en cours…",
  shared:
    "Position partagée. Elle est mise à jour toutes les 5 minutes tant que l'espace héros est ouvert.",
  denied:
    "Votre navigateur refuse la localisation : autorisez-la dans les réglages du site, puis rechargez la page.",
  unsupported: "Votre navigateur ne permet pas la géolocalisation.",
  error: "Position momentanément indisponible. Nouvel essai dans quelques minutes.",
};

/** Contenu du panneau « Partage de position » de « Mon profil » (le cadre est rendu par le serveur). */
export function LocationSharingPanel({ consentAt }: { consentAt: string | null }) {
  const [state, formAction] = useActionState(setLocationSharingAction, idleFormState);
  const { status, error, paused, pause, resume } = useHeroLocation();
  const [pausing, startPause] = useTransition();
  const sharing = consentAt !== null;

  return (
    <div className="space-y-4">
      <p className="text-sm text-muted">
        Partagez votre position pour être prévenu en priorité des incidents à moins de 10 km. Elle
        est arrondie à environ 110 m, remplacée à chaque envoi (aucun historique), supprimée au bout
        de 30 minutes et n&apos;est jamais affichée à personne. Elle n&apos;est envoyée que tant
        qu&apos;une page de l&apos;espace héros est ouverte.
      </p>

      {sharing && (
        <>
          <p className="text-sm">
            Partage activé le <time dateTime={consentAt}>{formatDate(consentAt)}</time>.
          </p>
          <p role="status" className="text-sm font-medium">
            {statusMessages[paused ? "off" : status]}
          </p>
          {error && (
            <p role="alert" className="text-sm text-danger">
              {error}
            </p>
          )}
          {paused ? (
            <button type="button" onClick={resume} className={buttonClasses("secondary")}>
              Reprendre
            </button>
          ) : (
            <button
              type="button"
              disabled={pausing}
              onClick={() => startPause(pause)}
              className={cn(buttonClasses("secondary"), "disabled:opacity-60")}
            >
              {pausing ? "Arrêt…" : "Arrêter pour l'instant"}
            </button>
          )}
        </>
      )}

      <form action={formAction} className="space-y-3">
        <input type="hidden" name="enabled" value={String(!sharing)} />
        <FormMessage state={state} />
        <SubmitButton variant={sharing ? "danger" : "primary"} pendingLabel="Enregistrement…">
          {sharing ? "Ne plus partager ma position" : "Partager ma position"}
        </SubmitButton>
      </form>
    </div>
  );
}
