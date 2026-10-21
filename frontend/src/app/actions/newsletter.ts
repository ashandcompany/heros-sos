"use server";

import { cookies } from "next/headers";
import {
  confirmNewsletterSubscription,
  subscribeToNewsletter,
  unsubscribeFromNewsletter,
} from "@/lib/api";
import { NEWSLETTER_COOKIE } from "@/lib/cookies";
import { apiFailureMessage, type TokenFormState } from "@/lib/validation/form-state";

export type NewsletterFormState =
  { status: "idle" } | { status: "success" } | { status: "error"; message: string; email: string };

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const INVALID_EMAIL = "Veuillez saisir une adresse email valide.";
const ONE_YEAR_IN_SECONDS = 60 * 60 * 24 * 365;

/**
 * Demande l'abonnement : l'adresse n'est abonnée qu'au clic sur le lien du mail de confirmation
 * (double opt-in). Le cookie qui masque le formulaire n'est donc posé qu'à la confirmation.
 */
export async function subscribeToNewsletterAction(
  _previousState: NewsletterFormState,
  formData: FormData,
): Promise<NewsletterFormState> {
  const email = String(formData.get("email") ?? "").trim();

  if (!EMAIL_PATTERN.test(email)) return { status: "error", message: INVALID_EMAIL, email };

  const result = await subscribeToNewsletter(email);
  if (!result.ok) {
    const message = result.error === "invalid-email" ? INVALID_EMAIL : apiFailureMessage(result);
    return { status: "error", message, email };
  }

  return { status: "success" };
}

export async function confirmNewsletterAction(
  _previousState: TokenFormState,
  formData: FormData,
): Promise<TokenFormState> {
  const result = await confirmNewsletterSubscription(String(formData.get("token") ?? ""));
  if (!result.ok) {
    return {
      status: "error",
      error: result.error === "invalid-token" ? "invalid-token" : "unavailable",
    };
  }

  (await cookies()).set(NEWSLETTER_COOKIE, "1", {
    maxAge: ONE_YEAR_IN_SECONDS,
    path: "/",
    sameSite: "lax",
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
  });

  return { status: "success" };
}

/**
 * Désabonne depuis le lien de désinscription d'une newsletter, sans connexion. Le cookie qui masque le
 * formulaire du pied de page est retiré : la personne peut se réabonner (avec un nouveau consentement).
 */
export async function unsubscribeNewsletterAction(
  _previousState: TokenFormState,
  formData: FormData,
): Promise<TokenFormState> {
  const result = await unsubscribeFromNewsletter(String(formData.get("token") ?? ""));
  if (!result.ok) {
    return {
      status: "error",
      error: result.error === "invalid-token" ? "invalid-token" : "unavailable",
    };
  }

  (await cookies()).delete(NEWSLETTER_COOKIE);
  return { status: "success" };
}
