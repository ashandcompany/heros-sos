"use server";

import { cookies } from "next/headers";
import { confirmNewsletterSubscription, subscribeToNewsletter } from "@/lib/api";
import { NEWSLETTER_COOKIE } from "@/lib/cookies";
import { apiFailureMessage } from "@/lib/validation/form-state";
import type { NewsletterConfirmationResult } from "@/types";

export type NewsletterFormState =
  { status: "idle" } | { status: "success" } | { status: "error"; message: string; email: string };

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const ONE_YEAR_IN_SECONDS = 60 * 60 * 24 * 365;
const INVALID_EMAIL = "Veuillez saisir une adresse email valide.";

export async function subscribeToNewsletterAction(
  _previousState: NewsletterFormState,
  formData: FormData,
): Promise<NewsletterFormState> {
  const email = String(formData.get("email") ?? "").trim();

  if (!EMAIL_PATTERN.test(email)) return { status: "error", message: INVALID_EMAIL, email };

  const result = await subscribeToNewsletter(email);

  if (!result.ok) {
    return {
      status: "error",
      message: result.error === "invalid-email" ? INVALID_EMAIL : apiFailureMessage(result),
      email,
    };
  }

  // Masque le formulaire : la demande est faite, il reste à cliquer sur le lien reçu par mail.
  (await cookies()).set(NEWSLETTER_COOKIE, "1", {
    maxAge: ONE_YEAR_IN_SECONDS,
    path: "/",
    sameSite: "lax",
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
  });

  return { status: "success" };
}

export type NewsletterConfirmationFormState =
  | { status: "idle" }
  | { status: "success" }
  | { status: "error"; error: Extract<NewsletterConfirmationResult, { ok: false }>["error"] };

export async function confirmNewsletterAction(
  _previousState: NewsletterConfirmationFormState,
  formData: FormData,
): Promise<NewsletterConfirmationFormState> {
  const result = await confirmNewsletterSubscription(String(formData.get("token") ?? ""));
  return result.ok ? { status: "success" } : { status: "error", error: result.error };
}
