import type { AuthUser, UserRole } from "@/types";
import { getSessionUser } from "./auth";

/**
 * Mock uniquement : retrouve l'utilisateur du jeton et vérifie son rôle, comme le fera l'API Go
 * (qui répondra 401 / 403). Les pages vérifient déjà la session avant d'appeler l'API.
 */
export async function userFromToken(token: string, role: UserRole): Promise<AuthUser> {
  const user = await getSessionUser(token);
  if (!user) throw new Error("401 : session invalide");
  if (user.role !== role) throw new Error("403 : accès refusé");
  return user;
}
