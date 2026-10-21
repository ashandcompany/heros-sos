import type { ApiFailure } from "./api";

export type NewsletterSubscriptionResult =
  { ok: true } | { ok: false; error: "invalid-email" } | ApiFailure;

export type NewsletterConfirmationResult =
  | { ok: true }
  /** Lien falsifié, expiré (48 h) ou devenu sans objet : l'API ne distingue pas ces cas. */
  | { ok: false; error: "invalid-token" | "unavailable" };
