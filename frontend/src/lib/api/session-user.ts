import { accountsMock } from "@/mocks/users";
import type { AuthUser, UserRole } from "@/types";
import { getSessionUser } from "./auth";
import { isApiEnabled } from "./config";

/**
 * Mock uniquement : retrouve l'utilisateur du jeton et vérifie son rôle, comme le fera l'API Go
 * (qui répondra 401 / 403). Les pages vérifient déjà la session avant d'appeler l'API.
 *
 * Quand l'API est configurée, la session est réelle mais les données de ces fonctions restent
 * mockées, et indexées par les identifiants des comptes mock (`user-1`, `hero-1`, `admin-1`) :
 * le vrai compte emprunte l'identifiant du compte mock de son rôle, en gardant son nom.
 */
export async function userFromToken(
  token: string,
  role: UserRole | readonly UserRole[],
): Promise<AuthUser> {
  const allowed: readonly UserRole[] = typeof role === "string" ? [role] : role;
  const user = await getSessionUser(token);
  if (!user) throw new Error("401 : session invalide");
  if (!allowed.includes(user.role)) throw new Error("403 : accès refusé");
  return asMockDataUser(user);
}

/**
 * Utilisateur vu par les données mockées : avec l'API, l'identifiant du compte mock de son rôle ;
 * sans l'API, l'utilisateur tel quel. Temporaire : à supprimer quand tous les domaines seront
 * branchés sur l'API.
 */
export function asMockDataUser(user: AuthUser): AuthUser {
  return isApiEnabled ? withMockIdentity(user) : user;
}

function withMockIdentity(user: AuthUser): AuthUser {
  const mockAccount = accountsMock.find((account) => account.user.role === user.role);
  return mockAccount ? { ...user, id: mockAccount.user.id } : user;
}
