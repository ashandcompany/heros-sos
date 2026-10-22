/*
 * Envoi d'une commande au serveur, par la route Next `/api/sync` : le navigateur ne parle pas directement à l'API
 * Go, la route y ajoute la session, le jeton CSRF et la clé d'idempotence.
 */

import type { SendableCommand, SendOutcome } from "./outbox.ts";

/** Chemin de la route Next qui relaie les commandes vers l'API. */
export const SYNC_ENDPOINT = "/api/sync";

/** Réponse de `/api/sync` en cas d'échec : le statut de l'API et son code d'erreur. */
interface SyncFailure {
  code?: string;
}

/** Ce que l'utilisateur lit quand le serveur refuse une action rejouée (code d'erreur de l'API → explication). */
const refusals: Record<string, string> = {
  intervention_already_active: "Un autre héros est déjà sur place : rejoignez son équipe.",
  hero_already_in_intervention: "Vous participez déjà à une autre intervention en cours.",
  incident_not_available: "Cet incident ne peut plus recevoir d'intervention.",
  intervention_participant_already_exists: "Ce super-héros est déjà lié à l'intervention.",
  intervention_participant_required: "Vous ne faites pas partie de l'équipe de cette intervention.",
  intervention_invitation_not_pending: "Aucune invitation en attente ne correspond.",
  intervention_outcome_not_pending: "Plus aucun résultat n'attend de validation.",
  intervention_outcome_already_proposed: "Un résultat est déjà en attente de validation.",
  intervention_outcome_already_validated: "Votre validation est déjà enregistrée.",
  intervention_version_conflict:
    "L'intervention a changé entre-temps : relisez-la avant de réessayer.",
  hero_not_approved: "Un profil de super-héros approuvé est requis.",
  validation_failed: "Certaines informations sont invalides.",
  forbidden: "Cette action ne vous est pas autorisée.",
  not_found: "La cible de cette action n'existe plus.",
};

const retryable = new Set([408, 425, 429, 500, 502, 503, 504]);

export function refusalMessage(code: string | undefined, status: number): string {
  return (code && refusals[code]) || `Refusée par le serveur (${code ?? `statut ${status}`}).`;
}

export async function sendCommand(
  command: SendableCommand,
  fetchImpl: typeof fetch = fetch,
): Promise<SendOutcome> {
  let response: Response;
  try {
    response = await fetchImpl(SYNC_ENDPOINT, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      credentials: "same-origin",
      body: JSON.stringify({
        commandId: command.id,
        path: command.path,
        body: command.body,
        createdAt: command.createdAt,
      }),
    });
  } catch {
    return { kind: "retry" };
  }

  if (response.ok) return { kind: "done" };
  if (response.status === 401) return { kind: "unauthenticated" };
  if (retryable.has(response.status)) return { kind: "retry" };
  const failure = (await response.json().catch(() => ({}))) as SyncFailure;
  // L'incident porte l'identifiant que l'appareil a choisi : s'il existe déjà, c'est que ce même envoi a déjà abouti
  // (réponse perdue en route). Rien à refuser, rien à refaire.
  if (failure.code === "incident_already_exists") return { kind: "done" };
  return { kind: "rejected", message: refusalMessage(failure.code, response.status) };
}
