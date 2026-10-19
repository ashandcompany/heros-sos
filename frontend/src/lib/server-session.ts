import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { SESSION_COOKIE } from "@/lib/cookies";
import { accountHomeByRole, routes } from "@/lib/routes";
import { getUserFromToken } from "@/lib/session";
import type { AuthUser } from "@/types";

/*
 * Helpers de session pour les Server Components et Server Actions.
 * Ils lisent les cookies : à appeler dans un composant rendu sous <Suspense>.
 */

/** Utilisateur connecté, ou `null` pour un visiteur. */
export async function getCurrentUser(): Promise<AuthUser | null> {
  return getUserFromToken((await cookies()).get(SESSION_COOKIE)?.value);
}

/**
 * Vérifie côté serveur que l'utilisateur est administrateur et renvoie son jeton.
 * Le proxy filtre déjà /admin, mais la doc Next recommande de revérifier au plus près des données.
 */
export async function requireAdminSession(): Promise<string> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  const user = await getUserFromToken(token);

  if (!user || !token) redirect(routes.login);
  if (user.role !== "admin") redirect(accountHomeByRole[user.role]);

  return token;
}
