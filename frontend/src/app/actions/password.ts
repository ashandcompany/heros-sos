"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { confirmPasswordReset, requestPasswordReset } from "@/lib/api";
import { SESSION_COOKIE, sessionCookieAttributes } from "@/lib/cookies";
import { getAccountHome } from "@/lib/session";
import { PASSWORD_RULES, validateNewPassword } from "@/lib/validation/signup";

export type ForgotPasswordFormState =
  { status: "idle" } | { status: "success" } | { status: "error"; message: string; email: string };

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export async function forgotPasswordAction(
  _previousState: ForgotPasswordFormState,
  formData: FormData,
): Promise<ForgotPasswordFormState> {
  // Le proxy protège déjà la page, mais une Server Action reste appelable directement en POST.
  const accountHome = await getAccountHome((await cookies()).get(SESSION_COOKIE)?.value);
  if (accountHome) redirect(accountHome);

  const email = String(formData.get("email") ?? "").trim();

  const invalidEmail = "Veuillez saisir une adresse email valide.";

  if (!EMAIL_PATTERN.test(email)) return { status: "error", message: invalidEmail, email };

  const result = await requestPasswordReset(email);
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

export type ResetPasswordFormState =
  | { status: "idle" }
  | { status: "success" }
  | { status: "invalid-token" }
  | {
      status: "error";
      fieldErrors: { password?: string; passwordConfirmation?: string };
      message?: string;
    };

export async function resetPasswordAction(
  _previousState: ResetPasswordFormState,
  formData: FormData,
): Promise<ResetPasswordFormState> {
  const token = String(formData.get("token") ?? "");
  const password = String(formData.get("password") ?? "");
  const passwordConfirmation = String(formData.get("passwordConfirmation") ?? "");

  const fieldErrors = {
    password: validateNewPassword(password),
    passwordConfirmation: !passwordConfirmation
      ? "Veuillez confirmer votre mot de passe."
      : passwordConfirmation !== password
        ? "Les mots de passe ne correspondent pas."
        : undefined,
  };
  if (fieldErrors.password || fieldErrors.passwordConfirmation) {
    return { status: "error", fieldErrors };
  }

  const result = await confirmPasswordReset(token, password);

  if (result.ok) {
    // L'API a fermé toutes les sessions du compte : le cookie éventuel ne vaut plus rien.
    (await cookies()).delete({ name: SESSION_COOKIE, ...sessionCookieAttributes });
    return { status: "success" };
  }

  switch (result.error) {
    case "invalid-token":
      return { status: "invalid-token" };
    case "weak-password":
      return {
        status: "error",
        fieldErrors: { password: `Ce mot de passe a été refusé. ${PASSWORD_RULES}` },
      };
    case "unavailable":
      return {
        status: "error",
        fieldErrors: {},
        message:
          "La réinitialisation est momentanément indisponible. Réessayez dans quelques instants.",
      };
  }
}
