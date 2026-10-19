"use server";

import { cookies } from "next/headers";
import { subscribeToNewsletter } from "@/lib/api";
import { NEWSLETTER_COOKIE } from "@/lib/cookies";

export type NewsletterFormState =
  { status: "idle" } | { status: "success" } | { status: "error"; message: string; email: string };

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const ONE_YEAR_IN_SECONDS = 60 * 60 * 24 * 365;

export async function subscribeToNewsletterAction(
  _previousState: NewsletterFormState,
  formData: FormData,
): Promise<NewsletterFormState> {
  const email = String(formData.get("email") ?? "").trim();

  if (!EMAIL_PATTERN.test(email)) {
    return { status: "error", message: "Veuillez saisir une adresse email valide.", email };
  }

  const result = await subscribeToNewsletter(email);

  if (!result.ok) {
    return {
      status: "error",
      message: "L'inscription a échoué. Veuillez réessayer plus tard.",
      email,
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
