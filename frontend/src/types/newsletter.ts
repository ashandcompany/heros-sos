import type { ApiFailure } from "./api";

/**
 * Double opt-in : `ok` signifie que la demande est enregistrée et qu'un lien de confirmation est
 * envoyé, pas que l'adresse est abonnée. L'API répond pareil pour toute adresse valide (inconnue,
 * en attente ou déjà abonnée) : elle ne révèle rien.
 */
export type NewsletterSubscriptionResult =
  { ok: true } | { ok: false; error: "invalid-email" } | ApiFailure;

export type NewsletterConfirmationResult =
  | { ok: true }
  /** Lien falsifié, expiré, ou devenu sans objet : l'API ne distingue pas ces cas. */
  | { ok: false; error: "invalid-token" }
  | ApiFailure;

export type NewsletterUnsubscribeResult =
  | { ok: true }
  /** Lien falsifié, ou fabriqué pour une autre action : l'API ne distingue pas ces cas. */
  | { ok: false; error: "invalid-token" }
  | ApiFailure;
