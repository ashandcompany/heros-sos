import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { asMockDataUser, getMyAvatar, type AvatarData } from "@/lib/api";
import { SESSION_COOKIE, sessionCookieAttributes } from "@/lib/cookies";
import { accountHomeByRole, routes } from "@/lib/routes";
import { getMfaRedirect, getUserFromToken } from "@/lib/session";
import type { AuthUser, UserRole } from "@/types";

/*
 * Helpers de session pour les Server Components et Server Actions.
 * Ils lisent les cookies : à appeler dans un composant rendu sous <Suspense>.
 */

/** Utilisateur connecté, ou `null` pour un visiteur. */
export async function getCurrentUser(): Promise<AuthUser | null> {
  return getUserFromToken((await cookies()).get(SESSION_COOKIE)?.value);
}

/** Photo de profil de l'utilisateur connecté, ou `null` (visiteur, ou compte sans profil). */
export async function getCurrentUserAvatar(): Promise<AvatarData | null> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  return token ? getMyAvatar(token) : null;
}

/**
 * Vérifie côté serveur que l'utilisateur a le rôle attendu (ou l'un d'eux) et renvoie son jeton
 * et son profil.
 * Le proxy filtre déjà les espaces protégés, mais la doc Next recommande de revérifier
 * au plus près des données (y compris dans chaque Server Action).
 */
export async function requireRole(
  role: UserRole | UserRole[],
): Promise<{ token: string; user: AuthUser }> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  const user = await getUserFromToken(token);

  if (!user || !token) redirect((await getMfaRedirect(token)) ?? routes.login);
  if (![role].flat().includes(user.role)) redirect(accountHomeByRole[user.role]);

  // Les données des espaces sont encore mockées : l'utilisateur renvoyé porte l'identifiant que
  // ces mocks connaissent (voir asMockDataUser).
  return { token, user: asMockDataUser(user) };
}

/**
 * Pose le cookie de session dans le navigateur, avec la durée de vie fixée par l'API : après la
 * connexion, et chaque fois que l'API remplace le jeton (activation ou validation de la 2FA).
 */
export async function setSessionCookie(token: string, expiresAt: string): Promise<void> {
  (await cookies()).set(SESSION_COOKIE, token, {
    ...sessionCookieAttributes,
    expires: new Date(expiresAt),
  });
}

export async function requireAdminSession(): Promise<string> {
  return (await requireRole("admin")).token;
}

export async function requireCitizenSession(): Promise<string> {
  return (await requireRole("citoyen")).token;
}

/**
 * Un compte super-héros n'existe qu'une fois sa demande validée par la modération
 * (voir `reviewHeroApplication`) : le rôle suffit à garantir un compte validé.
 */
export async function requireHeroSession(): Promise<{ token: string; user: AuthUser }> {
  return requireRole("super-heros");
}
