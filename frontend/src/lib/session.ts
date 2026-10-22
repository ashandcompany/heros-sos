import { getMfaSession, getSessionUser } from "@/lib/api";
import { accountHomeByRole, routes } from "@/lib/routes";
import type { AuthUser } from "@/types";

/** Utilisateur associé au jeton de session, ou `null` si absent / invalide. */
export async function getUserFromToken(sessionToken: string | undefined): Promise<AuthUser | null> {
  return sessionToken ? getSessionUser(sessionToken) : null;
}

/**
 * Route de l'espace personnel de l'utilisateur connecté (profil citoyen, espace héros
 * ou back-office), ou `null` si aucune session valide n'est associée au jeton.
 */
export async function getAccountHome(sessionToken: string | undefined): Promise<string | null> {
  const user = await getUserFromToken(sessionToken);
  return user ? accountHomeByRole[user.role] : null;
}

/**
 * Page de la double authentification où envoyer une session **limitée** (code à saisir, ou activation
 * obligatoire pas encore faite), ou `null` si la session n'est pas limitée ou n'existe pas. Une session
 * limitée est un visiteur partout ailleurs (voir `getSessionUser`) : sans cette redirection, un modérateur
 * qui vient de saisir son mot de passe serait renvoyé à la connexion en boucle.
 */
export async function getMfaRedirect(sessionToken: string | undefined): Promise<string | null> {
  const session = sessionToken ? await getMfaSession(sessionToken) : null;
  if (session?.status === "verification-required") return routes.loginVerification;
  if (session?.status === "enrollment-required") return routes.loginTwoFactorSetup;
  return null;
}
