import { accountsMock } from "@/mocks/users";
import type { AuthUser, LoginCredentials, LoginResult } from "@/types";
import { mockResponse } from "./config";

const MOCK_TOKEN_PREFIX = "mock-token-";

/**
 * Authentifie un utilisateur par email, ou par pseudo pour un super-héros.
 * Future implémentation : `POST ${API_URL}/auth/login` avec `{ identifier, password }`.
 */
export async function login({ identifier, password }: LoginCredentials): Promise<LoginResult> {
  const normalized = identifier.trim().toLowerCase();

  const account = accountsMock.find(
    ({ email, user }) =>
      email === normalized ||
      (user.role === "super-heros" && user.displayName.toLowerCase() === normalized),
  );

  if (!account || account.password !== password) {
    return mockResponse({ ok: false, error: "invalid-credentials" });
  }

  return mockResponse({
    ok: true,
    user: account.user,
    token: `${MOCK_TOKEN_PREFIX}${account.user.id}`,
  });
}

/**
 * Utilisateur associé à un jeton de session, ou `null` si le jeton est invalide ou expiré.
 * Future implémentation : `GET ${API_URL}/auth/me` avec `Authorization: Bearer ${token}`.
 */
export async function getSessionUser(token: string): Promise<AuthUser | null> {
  const userId = token.startsWith(MOCK_TOKEN_PREFIX) ? token.slice(MOCK_TOKEN_PREFIX.length) : "";
  const account = accountsMock.find(({ user }) => user.id === userId);

  return mockResponse(account?.user ?? null);
}

/**
 * Demande l'envoi d'un email de réinitialisation du mot de passe.
 * La réponse ne dit pas si l'adresse existe, pour ne pas révéler quels comptes sont inscrits.
 * Future implémentation : `POST ${API_URL}/auth/password-reset` avec `{ email }`.
 */
export async function requestPasswordReset(email: string): Promise<{ ok: boolean }> {
  void email;
  return mockResponse({ ok: true });
}
