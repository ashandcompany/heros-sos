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

/** Numéro de la newsletter rédigé par la modération (back-office). Brouillon tant que `sentAt` est `null`. */
export interface NewsletterIssue {
  id: string;
  subject: string;
  /** Corps en HTML, produit par l'éditeur du back-office et assaini par l'API à l'enregistrement. */
  body: string;
  /** Compte destinataire unique ; `null` pour tous les abonnés confirmés. */
  recipientUserId: string | null;
  /** Dates ISO 8601. */
  createdAt: string;
  /** Date d'envoi : le numéro est alors en lecture seule et ne repart jamais. */
  sentAt: string | null;
}

/** Numéro dans une liste : l'API n'y joint pas le corps. */
export type NewsletterIssueSummary = Omit<NewsletterIssue, "body">;

/** Champs saisis dans l'éditeur de newsletter. */
export interface NewsletterIssueInput {
  subject: string;
  body: string;
  recipientUserId: string | null;
}

export type NewsletterSendResult =
  /** `recipients` : nombre de mails confiés à la file d'envoi (la remise se fait en arrière-plan). */
  { ok: true; recipients: number } | { ok: false; error: string };
