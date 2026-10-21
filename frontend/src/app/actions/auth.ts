"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { login, logout, logoutEverywhere } from "@/lib/api";
import { SESSION_COOKIE, sessionCookieAttributes } from "@/lib/cookies";
import { formatRetryDelay } from "@/lib/format";
import { accountHomeByRole, routes } from "@/lib/routes";
import { apiFailureMessage, type FormState } from "@/lib/validation/form-state";
import type { LoginResult } from "@/types";

export interface LoginFormState {
  /** Identifiant renvoyé pour pré-remplir le champ (le mot de passe ne l'est jamais). */
  identifier: string;
  /** Case « Rester connecté », renvoyée pour rester cochée après une erreur. */
  rememberMe?: boolean;
  fieldErrors: Partial<Record<"identifier" | "password", string>>;
  formError?: string;
}

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const PSEUDO_PATTERN = /^[\p{L}\p{N} _-]{2,30}$/u;

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
  const rememberMe = formData.get("rememberMe") === "on";

  const fieldErrors: LoginFormState["fieldErrors"] = {
    identifier: validateIdentifier(identifier),
    password: password ? undefined : "Veuillez saisir votre mot de passe.",
  };

  if (fieldErrors.identifier || fieldErrors.password) {
    return { identifier, rememberMe, fieldErrors };
  }

  const result = await login({ identifier, password, rememberMe });

  if (!result.ok) {
    return { identifier, rememberMe, fieldErrors: {}, formError: loginErrorMessage(result) };
  }

  // Le cookie de l'API est recopié vers le navigateur avec la même durée de vie que la session.
  (await cookies()).set(SESSION_COOKIE, result.token, {
    ...sessionCookieAttributes,
    expires: new Date(result.expiresAt),
  });

  redirect(accountHomeByRole[result.user.role]);
}

/** Message affiché pour un échec de connexion, sans dire lequel des deux champs est faux. */
function loginErrorMessage(result: Extract<LoginResult, { ok: false }>): string {
  switch (result.error) {
    case "account-suspended":
      return "Ce compte a été suspendu par la modération. Contactez-nous pour en savoir plus.";
    case "account-not-activated":
      return "Votre compte n'est pas encore activé : consultez le mail d'activation.";
    case "too-many-attempts":
      return `Trop de tentatives de connexion. Réessayez dans ${formatRetryDelay(result.retryAfterSeconds)}.`;
    case "unavailable":
      return "La connexion est momentanément indisponible. Réessayez dans quelques instants.";
    case "invalid-credentials":
      return "Identifiant ou mot de passe incorrect.";
  }
}

export async function logoutAction(): Promise<void> {
  const cookieStore = await cookies();
  const token = cookieStore.get(SESSION_COOKIE)?.value;

  // Révoque la session côté API, puis efface le cookie dans tous les cas. Les attributs sont
  // repris : un cookie `__Host-` ne peut être effacé que par un Set-Cookie Secure.
  if (token) await logout(token);
  cookieStore.delete({ name: SESSION_COOKIE, ...sessionCookieAttributes });
  redirect(routes.home);
}

/**
 * Ferme toutes les sessions du compte (appareil perdu, mot de passe partagé…), celle-ci comprise,
 * puis renvoie vers la connexion. En cas d'échec, la session courante reste ouverte.
 */
export async function logoutEverywhereAction(): Promise<FormState> {
  const cookieStore = await cookies();
  const token = cookieStore.get(SESSION_COOKIE)?.value;

  if (token) {
    const result = await logoutEverywhere(token);
    if (!result.ok) return { status: "error", message: apiFailureMessage(result), fieldErrors: {} };
  }
  cookieStore.delete({ name: SESSION_COOKIE, ...sessionCookieAttributes });
  redirect(routes.login);
}
