import { headers } from "next/headers";
import { cache } from "react";
import { MOCK_SESSION_MAX_AGE } from "@/lib/cookies";
import { citizensMock } from "@/mocks/citizens";
import { accountsMock } from "@/mocks/users";
import type {
  AuthUser,
  LoginCredentials,
  LoginError,
  LoginResult,
  SignupPayload,
  SignupResult,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiUserView, apiFetch, fromApiUser, readSessionCookie } from "./http";

/*
 * Connexion, session et déconnexion sont branchées sur l'API Go dès que `API_URL` est renseignée
 * (sinon : mocks). Inscription et mot de passe oublié restent mockés.
 */

const MOCK_TOKEN_PREFIX = "mock-token-";

/** Compte suspendu par la modération (seuls les comptes citoyens sont concernés). */
const isSuspended = (userId: string) =>
  citizensMock.some(({ id, moderation }) => id === userId && moderation === "rejete");

/** Schéma OpenAPI : LoginResponse */
interface ApiLoginResponse {
  user: ApiUserView;
  csrfToken: string;
  expiresAt: string;
}

/** Schéma OpenAPI : MeResponse */
interface ApiMeResponse {
  user: ApiUserView;
  csrfToken: string;
}

/** Codes d'erreur de `POST /auth/login` → erreurs du frontend. */
const loginErrorByCode = new Map<string, LoginError>([
  ["invalid_credentials", "invalid-credentials"],
  // Identifiant mal formé (422) : même réponse que des identifiants faux.
  ["validation_failed", "invalid-credentials"],
  ["account_deactivated", "account-suspended"],
  ["account_not_activated", "account-not-activated"],
  ["account_locked", "too-many-attempts"],
  ["rate_limited", "too-many-attempts"],
]);

/**
 * Authentifie un utilisateur par email, ou par pseudo pour un super-héros.
 * API : `POST /auth/login` avec `{ identifier, password }`. Le cookie de session posé par l'API
 * est renvoyé dans `token`, à recopier vers le navigateur par l'action de connexion.
 */
export async function login(credentials: LoginCredentials): Promise<LoginResult> {
  return isApiEnabled ? loginWithApi(credentials) : loginWithMocks(credentials);
}

async function loginWithApi({ identifier, password }: LoginCredentials): Promise<LoginResult> {
  // L'API limite les tentatives par adresse IP et enregistre l'appareil de chaque session : on lui
  // transmet l'adresse du visiteur (posée par le reverse proxy) et son navigateur, sinon elle ne
  // verrait que le serveur Next.
  const incoming = await headers();
  const forwarded = new Headers();
  const forwardedFor = incoming.get("x-forwarded-for");
  const userAgent = incoming.get("user-agent");
  if (forwardedFor) forwarded.set("X-Forwarded-For", forwardedFor);
  if (userAgent) forwarded.set("User-Agent", userAgent);

  let response: Response;
  try {
    response = await apiFetch("/auth/login", {
      method: "POST",
      body: { identifier, password },
      headers: forwarded,
    });
  } catch (error) {
    const loginError = error instanceof ApiError ? loginErrorByCode.get(error.code) : undefined;
    if (loginError) {
      return {
        ok: false,
        error: loginError,
        retryAfterSeconds: (error as ApiError).retryAfterSeconds,
      };
    }
    console.error("Connexion : appel à l'API en échec", error);
    return { ok: false, error: "unavailable" };
  }

  const body = (await response.json()) as ApiLoginResponse;
  return {
    ok: true,
    user: fromApiUser(body.user),
    token: readSessionCookie(response),
    expiresAt: body.expiresAt,
  };
}

async function loginWithMocks({ identifier, password }: LoginCredentials): Promise<LoginResult> {
  const normalized = identifier.trim().toLowerCase();

  const account = accountsMock.find(
    ({ email, user }) =>
      email === normalized ||
      (user.role === "super-heros" && user.displayName.toLowerCase() === normalized),
  );

  if (!account || account.password !== password) {
    return mockResponse({ ok: false, error: "invalid-credentials" });
  }
  if (isSuspended(account.user.id)) {
    return mockResponse({ ok: false, error: "account-suspended" });
  }

  return mockResponse({
    ok: true,
    user: account.user,
    token: `${MOCK_TOKEN_PREFIX}${account.user.id}`,
    expiresAt: new Date(Date.now() + MOCK_SESSION_MAX_AGE * 1000).toISOString(),
  });
}

/**
 * Session de l'API (`GET /auth/me`), ou `null` si elle est absente, expirée ou révoquée (401).
 * Une API injoignable ou en erreur donne aussi `null` : sans réponse fiable, l'utilisateur est
 * traité en visiteur (les espaces protégés restent fermés).
 */
async function fetchApiSession(token: string): Promise<ApiMeResponse | null> {
  try {
    const response = await apiFetch("/auth/me", { sessionToken: token });
    return (await response.json()) as ApiMeResponse;
  } catch (error) {
    if (!(error instanceof ApiError && error.status === 401)) {
      console.error("Session : appel à l'API en échec", error);
    }
    return null;
  }
}

/**
 * Utilisateur associé à un jeton de session, ou `null` si le jeton est invalide, expiré
 * ou si le compte a été suspendu entre-temps.
 * API : `GET /auth/me`, avec le jeton dans le cookie de session. Mémorisé le temps d'un rendu
 * (`cache`) : les helpers de session et les fonctions mockées l'appellent plusieurs fois.
 */
export const getSessionUser = cache(async (token: string): Promise<AuthUser | null> => {
  if (isApiEnabled) {
    const session = await fetchApiSession(token);
    return session ? fromApiUser(session.user) : null;
  }

  const userId = token.startsWith(MOCK_TOKEN_PREFIX) ? token.slice(MOCK_TOKEN_PREFIX.length) : "";
  const account = isSuspended(userId)
    ? undefined
    : accountsMock.find(({ user }) => user.id === userId);

  return mockResponse(account?.user ?? null);
});

/**
 * Révoque la session côté API. Ne lève jamais d'erreur : l'action de déconnexion efface le cookie
 * du navigateur dans tous les cas.
 * API : `GET /auth/me` (pour le jeton CSRF), puis `POST /auth/logout` avec `X-CSRF-Token`.
 */
export async function logout(token: string): Promise<void> {
  if (!isApiEnabled) return;

  const session = await fetchApiSession(token);
  if (!session) return;

  try {
    await apiFetch("/auth/logout", {
      method: "POST",
      sessionToken: token,
      csrfToken: session.csrfToken,
    });
  } catch (error) {
    console.error("Déconnexion : révocation de la session par l'API en échec", error);
  }
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

/**
 * Crée un compte. Aucun rôle n'est choisi à l'inscription : le compte est créé citoyen
 * (le passage en super-héros relève d'un autre parcours, à définir avec l'API).
 * Reste mocké même quand l'API est configurée : le jeton mock renvoyé n'ouvre alors aucune
 * session (`GET /auth/me` le refuse), le nouveau compte reste visiteur.
 * Future implémentation : `POST ${API_URL}/auth/register` avec le `SignupPayload`.
 */
export async function register(payload: SignupPayload): Promise<SignupResult> {
  if (accountsMock.some(({ email }) => email === payload.email)) {
    return mockResponse({ ok: false, error: "email-already-used" });
  }

  const user: AuthUser = {
    id: crypto.randomUUID(),
    displayName: payload.firstName,
    role: "citoyen",
  };
  // Mock en mémoire : le compte reste utilisable jusqu'au redémarrage du serveur.
  accountsMock.push({ user, email: payload.email, password: payload.password });
  citizensMock.push({
    id: user.id,
    civility: payload.civility,
    firstName: payload.firstName,
    lastName: payload.lastName,
    birthDate: payload.birthDate,
    email: payload.email,
    pendingEmail: null,
    mobilePhone: payload.mobilePhone,
    address: { ...payload.address, complement: null },
    avatarUrl: null,
    newsletterSubscribed: false,
    registeredAt: new Date().toISOString(),
    moderation: "en-attente",
  });

  return mockResponse({ ok: true, user, token: `${MOCK_TOKEN_PREFIX}${user.id}` });
}
