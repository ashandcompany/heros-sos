import { redirect, unstable_rethrow } from "next/navigation";
import { cache } from "react";
import { SESSION_COOKIE } from "@/lib/cookies";
import { routes } from "@/lib/routes";
import type { ApiFailure, AuthUser, UserRole } from "@/types";
import { API_URL } from "./config";
import type { components } from "./schema";

/*
 * Client HTTP de l'API Go, utilisé par les fonctions de `lib/api/` déjà branchées.
 * Les appels partent du serveur Next : le navigateur ne parle jamais directement à l'API.
 */

/**
 * Schéma de l'OpenAPI, généré dans `schema.d.ts` par `npm run api:types`. Les réponses brutes de
 * l'API se typent avec : `ApiSchema<"LoginResponse">`, jamais avec une interface écrite à la main.
 */
export type ApiSchema<Name extends keyof components["schemas"]> = components["schemas"][Name];

/** Délai maximal d'un appel : au-delà, l'API est considérée comme indisponible. */
const API_TIMEOUT_MS = 10_000;

/** Erreur renvoyée par l'API (enveloppe `{ error: { code, message, fields } }`, schéma `Error`). */
export class ApiError extends Error {
  constructor(
    readonly status: number,
    /** Code stable (`invalid_credentials`, `rate_limited`…) ; `unknown` si la réponse n'en a pas. */
    readonly code: string,
    /** Délai annoncé par l'en-tête `Retry-After` (réponses 429). */
    readonly retryAfterSeconds?: number,
    /** Erreurs de validation (422) : champ (`email`, `address.city`…) → code (`invalid`, `weak`…). */
    readonly fields: Record<string, string> = {},
  ) {
    super(`API ${status} (${code})`);
    this.name = "ApiError";
  }
}

interface ApiRequest {
  method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
  /** Corps envoyé en JSON. */
  body?: unknown;
  /**
   * Corps envoyé tel quel (une image pour `PUT /me/avatar`), à la place de `body`. Son type se
   * donne dans `headers` (`Content-Type`).
   */
  rawBody?: Blob;
  /** Jeton de session : transmis dans le cookie que l'API attend, jamais dans `Authorization`. */
  sessionToken?: string;
  /**
   * Jeton CSRF de la session (`X-CSRF-Token`), exigé par l'API sur toute écriture authentifiée.
   * Inutile de le passer : il est lu dans la session (`getApiSession`, une fois par rendu).
   */
  csrfToken?: string;
  /**
   * Session expirée ou révoquée (401 sur un appel authentifié) : redirige vers la page de
   * connexion (par défaut), ou lève `ApiError` (`false`) pour un appel qui gère ce cas lui-même.
   */
  redirectOnUnauthorized?: boolean;
  headers?: HeadersInit;
}

/**
 * Appelle l'API et renvoie la réponse si son statut est 2xx. Lève `ApiError` pour une réponse
 * d'erreur, et l'erreur réseau d'origine (`TypeError`, `TimeoutError`) si l'API est injoignable.
 *
 * Un 401 sur un appel authentifié redirige vers la connexion (`redirect()` de Next) : dans un
 * `catch`, appeler `toApiFailure` (ou `unstable_rethrow`) pour ne pas intercepter la redirection.
 */
export async function apiFetch(path: string, request: ApiRequest = {}): Promise<Response> {
  const method = request.method ?? "GET";
  const { sessionToken } = request;
  const csrfToken =
    request.csrfToken ??
    (sessionToken && method !== "GET" ? (await getApiSession(sessionToken))?.csrfToken : undefined);

  const headers = new Headers(request.headers);
  headers.set("Accept", "application/json");
  if (request.body !== undefined) headers.set("Content-Type", "application/json");
  if (sessionToken) headers.set("Cookie", `${SESSION_COOKIE}=${sessionToken}`);
  if (csrfToken) headers.set("X-CSRF-Token", csrfToken);

  const response = await fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: request.body === undefined ? request.rawBody : JSON.stringify(request.body),
    cache: "no-store",
    signal: AbortSignal.timeout(API_TIMEOUT_MS),
  });

  if (response.status === 401 && sessionToken && request.redirectOnUnauthorized !== false) {
    redirect(routes.login);
  }
  if (!response.ok) throw await toApiError(response);
  return response;
}

async function toApiError(response: Response): Promise<ApiError> {
  const body: unknown = await response.json().catch(() => null);
  const error =
    typeof body === "object" &&
    body !== null &&
    "error" in body &&
    typeof body.error === "object" &&
    body.error !== null
      ? body.error
      : {};
  const code = "code" in error && typeof error.code === "string" ? error.code : "unknown";
  const fields =
    "fields" in error && typeof error.fields === "object" && error.fields !== null
      ? Object.fromEntries(
          Object.entries(error.fields).filter(
            (entry): entry is [string, string] => typeof entry[1] === "string",
          ),
        )
      : {};
  const retryAfter = Number(response.headers.get("Retry-After"));

  return new ApiError(
    response.status,
    code,
    Number.isFinite(retryAfter) && retryAfter > 0 ? retryAfter : undefined,
    fields,
  );
}

/**
 * Traduit une erreur d'appel en échec commun (403, 413, 415, 429, API indisponible). À appeler
 * dans le `catch` d'une fonction de `lib/api/`, **après** avoir traité ses codes métier (certains
 * sont des 403 ou des 429, comme `hero_request_not_eligible` ou `account_locked`) :
 *
 * ```ts
 * } catch (error) {
 *   if (error instanceof ApiError && error.code === "hero_request_pending") {
 *     return { ok: false, error: "already-pending" };
 *   }
 *   return toApiFailure(error, "Demande de statut super-héros");
 * }
 * ```
 *
 * Relance la redirection d'une session expirée (401) au lieu de l'intercepter.
 */
export function toApiFailure(error: unknown, context: string): ApiFailure {
  unstable_rethrow(error);

  if (error instanceof ApiError) {
    switch (error.status) {
      case 403:
        // Origine ou jeton CSRF refusés, ou rôle insuffisant : les pages vérifiant déjà le rôle,
        // c'est le signe d'un bug à corriger, pas d'une action du visiteur.
        console.warn(`${context} : accès refusé par l'API (${error.code})`);
        return { ok: false, error: "forbidden" };
      case 413:
        return { ok: false, error: "payload-too-large" };
      case 415:
        return { ok: false, error: "unsupported-media-type" };
      case 429:
        return {
          ok: false,
          error: "too-many-attempts",
          retryAfterSeconds: error.retryAfterSeconds,
        };
    }
  }

  console.error(`${context} : appel à l'API en échec`, error);
  return { ok: false, error: "unavailable" };
}

/**
 * Session de l'API (`GET /auth/me`) : compte connecté et jeton CSRF. `null` si elle est absente,
 * expirée ou révoquée (401) ; une API injoignable ou en erreur donne aussi `null`. Mémorisée le
 * temps d'un rendu ou d'une action (`cache`) : la page, les helpers de session et chaque écriture
 * la relisent sans nouvel appel.
 */
export const getApiSession = cache(
  async (sessionToken: string): Promise<ApiSchema<"MeResponse"> | null> => {
    try {
      const response = await apiFetch("/auth/me", {
        sessionToken,
        redirectOnUnauthorized: false,
      });
      return (await response.json()) as ApiSchema<"MeResponse">;
    } catch (error) {
      if (!(error instanceof ApiError && error.status === 401)) {
        console.error("Session : appel à l'API en échec", error);
      }
      return null;
    }
  },
);

/**
 * Valeur du cookie de session posé par l'API dans sa réponse (`Set-Cookie`). Le frontend le
 * recopie tel quel vers le navigateur : il doit donc porter le même nom des deux côtés.
 */
export function readSessionCookie(response: Response): string {
  const prefix = `${SESSION_COOKIE}=`;
  const header = response.headers.getSetCookie().find((cookie) => cookie.startsWith(prefix));

  if (!header) {
    throw new Error(
      `L'API n'a pas posé de cookie « ${SESSION_COOKIE} ». Le cookie Secure du backend ` +
        "(HEROS_COOKIE_SECURE) doit correspondre au mode du frontend : voir docs/backend-contract.md.",
    );
  }
  return header.slice(prefix.length).split(";", 1)[0];
}

type ApiRole = ApiSchema<"UserView">["role"];

const roleFromApi: Record<ApiRole, UserRole> = {
  citizen: "citoyen",
  hero: "super-heros",
  moderator: "admin",
};

const roleToApiByRole: Record<UserRole, ApiRole> = {
  citoyen: "citizen",
  "super-heros": "hero",
  admin: "moderator",
};

/** Rôle de l'API vers celui du frontend : seul endroit où les rôles de l'API apparaissent. */
export function fromApiRole(role: ApiRole): UserRole {
  const converted = roleFromApi[role];
  if (!converted) throw new Error(`Rôle inconnu renvoyé par l'API : ${role}`);
  return converted;
}

/** Rôle du frontend vers celui de l'API (filtres). */
export function roleToApi(role: UserRole): ApiRole {
  return roleToApiByRole[role];
}

/** Convertit un compte de l'API vers `AuthUser`. */
export function fromApiUser(user: ApiSchema<"UserView">): AuthUser {
  return { id: user.id, displayName: user.displayName, role: fromApiRole(user.role) };
}
