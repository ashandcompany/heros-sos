import { getSessionUser } from "@/lib/api";
import { accountHomeByRole } from "@/lib/routes";

/**
 * Route de l'espace personnel de l'utilisateur connecté (profil citoyen ou espace héros),
 * ou `null` si aucune session valide n'est associée au jeton.
 */
export async function getAccountHome(sessionToken: string | undefined): Promise<string | null> {
  if (!sessionToken) return null;

  const user = await getSessionUser(sessionToken);
  return user ? accountHomeByRole[user.role] : null;
}
