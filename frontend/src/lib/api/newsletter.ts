import type { NewsletterConfirmationResult, NewsletterSubscriptionResult } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure, visitorHeaders } from "./http";

/**
 * Demande l'abonnement d'une adresse à la newsletter, en deux temps : l'API envoie un lien de
 * confirmation (`routes.newsletterConfirmation`) et n'envoie rien avant le clic. La réponse est
 * la même pour toute adresse valide (déjà abonnée ou non).
 * API : `POST /newsletter/subscriptions` avec `{ email }`, limité par IP et par adresse.
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
    return toApiFailure(error, "Abonnement à la newsletter");
  }
}

/**
 * Confirme l'abonnement avec le jeton du lien reçu par mail (valable 48 h ; un second clic réussit
 * aussi). API : `POST /newsletter/subscriptions/confirm` avec `{ token }`. Mock : tout jeton non
 * vide est accepté.
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
    // Jeton invalide (`invalid_token`) ou mal formé (`bad_request`) : même réponse pour le visiteur.
    if (error instanceof ApiError && error.status === 400) {
      return { ok: false, error: "invalid-token" };
    }
    console.error("Confirmation de la newsletter : appel à l'API en échec", error);
    return { ok: false, error: "unavailable" };
  }
}
