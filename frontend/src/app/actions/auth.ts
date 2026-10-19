"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { login } from "@/lib/api";
import { SESSION_COOKIE } from "@/lib/cookies";
import { routes } from "@/lib/routes";

export interface LoginFormState {
  /** Identifiant renvoyé pour pré-remplir le champ (le mot de passe ne l'est jamais). */
  identifier: string;
  fieldErrors: Partial<Record<"identifier" | "password", string>>;
  formError?: string;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const PSEUDO_PATTERN = /^[\p{L}\p{N} _-]{2,30}$/u;
const ONE_WEEK_IN_SECONDS = 60 * 60 * 24 * 7;

function validateIdentifier(identifier: string): string | undefined {
  if (!identifier) return "Veuillez saisir votre adresse email ou votre pseudo.";
  if (identifier.includes("@")) {
    return EMAIL_PATTERN.test(identifier) ? undefined : "L'adresse email n'est pas valide.";
  }
  return PSEUDO_PATTERN.test(identifier) ? undefined : "Le pseudo n'est pas valide.";
}

export async function loginAction(
  _previousState: LoginFormState,
  formData: FormData,
): Promise<LoginFormState> {
  const identifier = String(formData.get("identifier") ?? "").trim();
  const password = String(formData.get("password") ?? "");

  const fieldErrors: LoginFormState["fieldErrors"] = {
    identifier: validateIdentifier(identifier),
    password: password ? undefined : "Veuillez saisir votre mot de passe.",
  };

  if (fieldErrors.identifier || fieldErrors.password) {
    return { identifier, fieldErrors };
  }

  const result = await login({ identifier, password });

  if (!result.ok) {
    return {
      identifier,
      fieldErrors: {},
      formError: "Identifiant ou mot de passe incorrect.",
    };
  }

  // Placeholder : le format de session sera fixé avec l'API Go (JWT, session opaque…).
  (await cookies()).set(SESSION_COOKIE, result.token, {
    maxAge: ONE_WEEK_IN_SECONDS,
    path: "/",
    sameSite: "lax",
    httpOnly: true,
    secure: process.env.NODE_ENV === "production",
  });

  redirect(routes.home);
}
