"use server";

import { activateAccount, resendActivation } from "@/lib/api";
import type { ActivationResult } from "@/types";

export type ActivationFormState =
  | { status: "idle" }
  | { status: "success" }
  | { status: "error"; error: Extract<ActivationResult, { ok: false }>["error"] };

export async function activateAccountAction(
  _previousState: ActivationFormState,
  formData: FormData,
): Promise<ActivationFormState> {
  const result = await activateAccount(String(formData.get("token") ?? ""));
  return result.ok ? { status: "success" } : { status: "error", error: result.error };
}

export type ResendActivationFormState =
  { status: "idle" } | { status: "success" } | { status: "error"; message: string; email: string };

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export async function resendActivationAction(
  _previousState: ResendActivationFormState,
  formData: FormData,
): Promise<ResendActivationFormState> {
  const email = String(formData.get("email") ?? "").trim();
  const invalidEmail = "Veuillez saisir une adresse email valide.";

  if (!EMAIL_PATTERN.test(email)) return { status: "error", message: invalidEmail, email };

  const result = await resendActivation(email);
  if (result.ok) return { status: "success" };

  switch (result.error) {
    case "invalid-email":
      return { status: "error", message: invalidEmail, email };
    case "too-many-attempts": {
      const minutes = Math.ceil((result.retryAfterSeconds ?? 60) / 60);
      return {
        status: "error",
        message: `Trop de demandes. Réessayez dans ${minutes} minute${minutes > 1 ? "s" : ""}.`,
        email,
      };
    }
    case "unavailable":
      return {
        status: "error",
        message: "La demande n'a pas pu aboutir. Veuillez réessayer plus tard.",
        email,
      };
  }
}
