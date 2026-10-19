import { accountsMock } from "@/mocks/users";
import type { LoginCredentials, LoginResult } from "@/types";
import { mockResponse } from "./config";

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

  return mockResponse({ ok: true, user: account.user, token: `mock-token-${account.user.id}` });
}
