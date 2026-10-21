import { cache } from "react";
import { MOCK_REMEMBERED_SESSION_MAX_AGE, MOCK_SESSION_MAX_AGE } from "@/lib/cookies";
import { citizensMock } from "@/mocks/citizens";
import { accountsMock } from "@/mocks/users";
import type {
  ActivationResult,
  AuthUser,
  LoginCredentials,
  LoginError,
  LoginResult,
  LogoutAllResult,
  MfaStatus,
  PasswordResetConfirmResult,
  PasswordResetRequestResult,
  ResendActivationResult,
  SignupPayload,
  SignupResult,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import {
  ApiError,
  type ApiSchema,
  apiFetch,
  fromApiMfaStatus,
  fromApiUser,
  getApiSession,
  isLimitedSession,
  readSessionCookie,
  toApiFailure,
  visitorHeaders,
} from "./http";

/*
 * Inscription, activation, connexion, session, déconnexion et mot de passe oublié sont branchés sur
 * l'API Go dès que `API_URL` est renseignée (sinon : mocks).
 */

const MOCK_TOKEN_PREFIX = "mock-token-";

/** Compte suspendu par la modération (seuls les comptes citoyens sont concernés). */
const isSuspended = (userId: string) =>
  citizensMock.some(({ id, moderation }) => id === userId && moderation === "rejete");

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

async function loginWithApi({
  identifier,
  password,
  rememberMe = false,
}: LoginCredentials): Promise<LoginResult> {
  let response: Response;
  try {
    response = await apiFetch("/auth/login", {
      method: "POST",
      body: { identifier, password, rememberMe } satisfies ApiSchema<"LoginRequest">,
      headers: await visitorHeaders(),
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

  const body = (await response.json()) as ApiSchema<"LoginResponse">;
  return {
    ok: true,
    user: fromApiUser(body.user),
    token: readSessionCookie(response),
    expiresAt: body.expiresAt,
    mfa: fromApiMfaStatus(body.mfa),
  };
}

async function loginWithMocks({
  identifier,
  password,
  rememberMe = false,
}: LoginCredentials): Promise<LoginResult> {
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

  // Comme l'API : « Rester connecté » est sans effet pour un modérateur.
  const maxAge =
    rememberMe && account.user.role !== "admin"
      ? MOCK_REMEMBERED_SESSION_MAX_AGE
      : MOCK_SESSION_MAX_AGE;
  return mockResponse({
    ok: true,
    user: account.user,
    token: `${MOCK_TOKEN_PREFIX}${account.user.id}`,
    expiresAt: new Date(Date.now() + maxAge * 1000).toISOString(),
    mfa: "not-enrolled",
  });
}

/**
 * Utilisateur associé à un jeton de session, ou `null` si le jeton est invalide, expiré ou si le compte
 * a été suspendu entre-temps. Une API injoignable donne aussi `null` : sans réponse fiable, l'utilisateur
 * est traité en visiteur (les espaces protégés restent fermés).
 *
 * Une session **limitée** (code de double authentification à saisir, ou activation obligatoire pas encore
 * faite) donne aussi `null` : elle n'ouvre rien d'autre que la 2FA elle-même, et le frontend ne doit jamais
 * l'afficher comme connectée. Les pages de la 2FA la lisent avec `getMfaSession`.
 * API : `GET /auth/me` (`getApiSession`), avec le jeton dans le cookie de session. Mémorisé le
 * temps d'un rendu (`cache`) : les helpers de session et les fonctions mockées l'appellent
 * plusieurs fois, et les écritures y relisent le jeton CSRF.
 */
export const getSessionUser = cache(async (token: string): Promise<AuthUser | null> => {
  if (isApiEnabled) {
    const session = await getApiSession(token);
    if (!session || isLimitedSession(fromApiMfaStatus(session.mfa))) return null;
    return fromApiUser(session.user);
  }

  const userId = token.startsWith(MOCK_TOKEN_PREFIX) ? token.slice(MOCK_TOKEN_PREFIX.length) : "";
  const account = isSuspended(userId)
    ? undefined
    : accountsMock.find(({ user }) => user.id === userId);

  return mockResponse(account?.user ?? null);
});

/**
 * Compte et état de la double authentification d'une session, **limitée ou non** : c'est ce que lisent les
 * pages de la 2FA (saisie du code, activation) et ce qui décide où envoyer une session limitée. `null` si
 * la session est invalide. Sans API, aucune 2FA n'existe : la session est toujours complète.
 */
export const getMfaSession = cache(
  async (token: string): Promise<{ user: AuthUser; status: MfaStatus } | null> => {
    if (!isApiEnabled) {
      const user = await getSessionUser(token);
      return user ? { user, status: "not-enrolled" } : null;
    }
    const session = await getApiSession(token);
    return session
      ? { user: fromApiUser(session.user), status: fromApiMfaStatus(session.mfa) }
      : null;
  },
);

/**
 * Révoque la session côté API. Ne lève jamais d'erreur : l'action de déconnexion efface le cookie
 * du navigateur dans tous les cas.
 * API : `GET /auth/me` (pour le jeton CSRF), puis `POST /auth/logout` avec `X-CSRF-Token`.
 */
export async function logout(token: string): Promise<void> {
  if (!isApiEnabled) return;

  // Session déjà expirée ou révoquée : rien à révoquer.
  const session = await getApiSession(token);
  if (!session) return;

  try {
    await apiFetch("/auth/logout", {
      method: "POST",
      sessionToken: token,
      csrfToken: session.csrfToken,
      redirectOnUnauthorized: false,
    });
  } catch (error) {
    console.error("Déconnexion : révocation de la session par l'API en échec", error);
  }
}

/**
 * Ferme toutes les sessions du compte, sur tous ses appareils, celle-ci comprise. Contrairement à
 * `logout`, un échec est rapporté : la personne doit savoir que ses autres appareils sont peut-être
 * restés connectés. Mock : sans effet (un jeton mock est sans état, il ne se révoque pas).
 * API : `POST /auth/logout-all` (le jeton CSRF est ajouté par `apiFetch`).
 */
export async function logoutAll(token: string): Promise<LogoutAllResult> {
  if (!isApiEnabled) return mockResponse({ ok: true });

  try {
    await apiFetch("/auth/logout-all", {
      method: "POST",
      sessionToken: token,
      redirectOnUnauthorized: false,
    });
    return { ok: true };
  } catch (error) {
    // Une session déjà fermée (401) est le résultat voulu : l'action efface le cookie ensuite.
    if (error instanceof ApiError && error.status === 401) return { ok: true };
    return toApiFailure(error, "Déconnexion de tous les appareils");
  }
}

/**
 * Active un compte avec le jeton du lien reçu par mail (valable 24 h, utilisable une fois).
 * API : `POST /auth/activate` avec `{ token }`. Mock : tout jeton non vide est accepté.
 */
export async function activateAccount(token: string): Promise<ActivationResult> {
  if (!isApiEnabled) {
    return mockResponse(token ? { ok: true } : { ok: false, error: "invalid-token" });
  }

  try {
    await apiFetch("/auth/activate", { method: "POST", body: { token } });
    return { ok: true };
  } catch (error) {
    // Jeton invalide (`invalid_token`) ou mal formé (`bad_request`) : même réponse pour le visiteur.
    if (error instanceof ApiError && error.status === 400) {
      return { ok: false, error: "invalid-token" };
    }
    console.error("Activation : appel à l'API en échec", error);
    return { ok: false, error: "unavailable" };
  }
}

/**
 * Renvoie un lien d'activation (les précédents sont invalidés). La réponse ne dit pas si l'adresse
 * correspond à un compte en attente, pour ne pas révéler quels comptes sont inscrits.
 * API : `POST /auth/resend-activation` avec `{ email }`.
 */
export async function resendActivation(email: string): Promise<ResendActivationResult> {
  if (!isApiEnabled) return mockResponse({ ok: true });

  try {
    await apiFetch("/auth/resend-activation", {
      method: "POST",
      body: { email },
      headers: await visitorHeaders(),
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "validation_failed") return { ok: false, error: "invalid-email" };
      if (error.code === "rate_limited") {
        return {
          ok: false,
          error: "too-many-attempts",
          retryAfterSeconds: error.retryAfterSeconds,
        };
      }
    }
    console.error("Renvoi du lien d'activation : appel à l'API en échec", error);
    return { ok: false, error: "unavailable" };
  }
}

/**
 * Demande l'envoi d'un email de réinitialisation du mot de passe (les liens précédents sont
 * invalidés). La réponse ne dit pas si l'adresse existe, pour ne pas révéler quels comptes sont
 * inscrits. API : `POST /auth/password-reset` avec `{ email }`.
 */
export async function requestPasswordReset(email: string): Promise<PasswordResetRequestResult> {
  if (!isApiEnabled) return mockResponse({ ok: true });

  try {
    await apiFetch("/auth/password-reset", {
      method: "POST",
      body: { email },
      headers: await visitorHeaders(),
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "validation_failed") return { ok: false, error: "invalid-email" };
      if (error.code === "rate_limited") {
        return {
          ok: false,
          error: "too-many-attempts",
          retryAfterSeconds: error.retryAfterSeconds,
        };
      }
    }
    console.error("Mot de passe oublié : appel à l'API en échec", error);
    return { ok: false, error: "unavailable" };
  }
}

/**
 * Choisit un nouveau mot de passe avec le jeton du lien reçu par mail (valable 1 h, utilisable une
 * fois). L'API ferme alors toutes les sessions du compte.
 * API : `POST /auth/password-reset/confirm` avec `{ token, newPassword }`. Mock : tout jeton non
 * vide est accepté.
 */
export async function confirmPasswordReset(
  token: string,
  newPassword: string,
): Promise<PasswordResetConfirmResult> {
  if (!isApiEnabled) {
    return mockResponse(token ? { ok: true } : { ok: false, error: "invalid-token" });
  }

  try {
    await apiFetch("/auth/password-reset/confirm", {
      method: "POST",
      body: { token, newPassword },
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      // Jeton invalide (`invalid_token`) ou mal formé (`bad_request`) : même réponse pour le visiteur.
      if (error.status === 400) return { ok: false, error: "invalid-token" };
      // Seul champ validé par l'API à part le jeton : le mot de passe trop faible ou trop long.
      if (error.code === "validation_failed") return { ok: false, error: "weak-password" };
    }
    console.error("Réinitialisation du mot de passe : appel à l'API en échec", error);
    return { ok: false, error: "unavailable" };
  }
}

/**
 * Crée un compte citoyen en attente d'activation : aucune session n'est ouverte, l'API envoie un
 * mail contenant le lien d'activation (`routes.activation`). Aucun rôle n'est choisi à
 * l'inscription : le statut de super-héros se demande ensuite (`POST /heroes/requests`).
 * API : `POST /auth/register` avec le `SignupPayload`.
 */
export async function register(payload: SignupPayload): Promise<SignupResult> {
  if (!isApiEnabled) return registerWithMocks(payload);

  try {
    const response = await apiFetch("/auth/register", {
      method: "POST",
      body: payload,
      headers: await visitorHeaders(),
    });
    const body = (await response.json()) as ApiSchema<"RegisterResponse">;
    return { ok: true, user: fromApiUser(body.user) };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "email_already_used") return { ok: false, error: "email-already-used" };
      if (error.code === "validation_failed") {
        return { ok: false, error: "invalid-fields", fields: error.fields };
      }
      if (error.code === "rate_limited") {
        return {
          ok: false,
          error: "too-many-attempts",
          retryAfterSeconds: error.retryAfterSeconds,
        };
      }
    }
    console.error("Inscription : appel à l'API en échec", error);
    return { ok: false, error: "unavailable" };
  }
}

/**
 * Mock en mémoire : le compte reste utilisable jusqu'au redémarrage du serveur. Comme avec l'API,
 * aucune session n'est ouverte (le mock de connexion ne vérifie pas l'activation).
 */
async function registerWithMocks(payload: SignupPayload): Promise<SignupResult> {
  if (accountsMock.some(({ email }) => email === payload.email)) {
    return mockResponse({ ok: false, error: "email-already-used" });
  }

  const user: AuthUser = {
    id: crypto.randomUUID(),
    displayName: payload.firstName,
    role: "citoyen",
  };
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

  return mockResponse({ ok: true, user });
}
