import type {
  NewsletterConfirmationResult,
  NewsletterSubscriptionResult,
  NewsletterUnsubscribeResult,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure, visitorHeaders } from "./http";

/**
 * Demande l'abonnement d'une adresse à la newsletter, en deux temps (double opt-in) : l'API envoie
 * un lien de confirmation (`routes.newsletterConfirmation`, valable 48 h) et n'envoie rien avant le
 * clic. Sa réponse est la même pour toute adresse valide : elle ne dit pas si l'adresse est déjà
 * abonnée. Limité par adresse IP et par adresse email.
 * API : `POST /newsletter/subscriptions` avec `{ email }`.
 */
export async function subscribeToNewsletter(email: string): Promise<NewsletterSubscriptionResult> {
  if (!isApiEnabled) return mockResponse({ ok: true });

  try {
    await apiFetch("/newsletter/subscriptions", {
      method: "POST",
      body: { email } satisfies ApiSchema<"NewsletterSubscriptionRequest">,
      headers: await visitorHeaders(),
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError && error.code === "validation_failed") {
      return { ok: false, error: "invalid-email" };
    }
    return toApiFailure(error, "Inscription à la newsletter");
  }
}

/**
 * Confirme l'abonnement avec le jeton du lien reçu par mail : la date de confirmation est la preuve
 * du consentement. Cliquer deux fois sur le lien réussit les deux fois.
 * API : `POST /newsletter/subscriptions/confirm` avec `{ token }`. Mock : tout jeton non vide est
 * accepté.
 */
export async function confirmNewsletterSubscription(
  token: string,
): Promise<NewsletterConfirmationResult> {
  if (!isApiEnabled) {
    return mockResponse(token ? { ok: true } : { ok: false, error: "invalid-token" });
  }

  try {
    await apiFetch("/newsletter/subscriptions/confirm", {
      method: "POST",
      body: { token } satisfies ApiSchema<"TokenRequest">,
    });
    return { ok: true };
  } catch (error) {
    // Lien invalide (`invalid_token`) ou mal formé (`bad_request`) : même réponse pour le visiteur.
    if (error instanceof ApiError && error.status === 400) {
      return { ok: false, error: "invalid-token" };
    }
    return toApiFailure(error, "Confirmation de l'abonnement à la newsletter");
  }
}

/**
 * Désabonne l'abonné désigné par le lien de désinscription que porte chaque newsletter, sans connexion. Le lien
 * n'expire pas : celui d'une ancienne newsletter reste utilisable. Cliquer deux fois réussit. Se réabonner
 * depuis le pied de page redemande un consentement.
 * API : `POST /newsletter/subscriptions/unsubscribe` avec `{ token }`. Mock : tout jeton non vide est accepté.
 */
export async function unsubscribeFromNewsletter(
  token: string,
): Promise<NewsletterUnsubscribeResult> {
  if (!isApiEnabled) {
    return mockResponse(token ? { ok: true } : { ok: false, error: "invalid-token" });
  }

  try {
    await apiFetch("/newsletter/subscriptions/unsubscribe", {
      method: "POST",
      body: { token } satisfies ApiSchema<"TokenRequest">,
    });
    return { ok: true };
  } catch (error) {
    // Lien invalide (`invalid_token`) ou mal formé (`bad_request`) : même réponse pour le visiteur.
    if (error instanceof ApiError && error.status === 400) {
      return { ok: false, error: "invalid-token" };
    }
    return toApiFailure(error, "Désinscription de la newsletter");
  }
}
