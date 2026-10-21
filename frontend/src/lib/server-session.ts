import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { getMyAvatar, type AvatarData } from "@/lib/api";
import { SESSION_COOKIE } from "@/lib/cookies";
import { accountHomeByRole, routes } from "@/lib/routes";
import { getUserFromToken } from "@/lib/session";
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

  if (!user || !token) redirect(routes.login);
  if (![role].flat().includes(user.role)) redirect(accountHomeByRole[user.role]);

  return { token, user };
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
