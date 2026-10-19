import { getSessionUser } from "@/lib/api";
import { accountHomeByRole } from "@/lib/routes";
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
