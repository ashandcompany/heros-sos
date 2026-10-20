import { SESSION_COOKIE } from "@/lib/cookies";
import type { AuthUser, UserRole } from "@/types";
import { API_URL } from "./config";

/*
 * Client HTTP de l'API Go, utilisé par les fonctions de `lib/api/` déjà branchées.
 * Les appels partent du serveur Next : le navigateur ne parle jamais directement à l'API.
 */

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
  /** Jeton de session : transmis dans le cookie que l'API attend, jamais dans `Authorization`. */
  sessionToken?: string;
  /** Jeton CSRF de la session (`X-CSRF-Token`), exigé par l'API sur toute écriture authentifiée. */
  csrfToken?: string;
  headers?: HeadersInit;
}

/**
 * Appelle l'API et renvoie la réponse si son statut est 2xx. Lève `ApiError` pour une réponse
 * d'erreur, et l'erreur réseau d'origine (`TypeError`, `TimeoutError`) si l'API est injoignable.
 */
export async function apiFetch(path: string, request: ApiRequest = {}): Promise<Response> {
  const headers = new Headers(request.headers);
  headers.set("Accept", "application/json");
  if (request.body !== undefined) headers.set("Content-Type", "application/json");
  if (request.sessionToken) headers.set("Cookie", `${SESSION_COOKIE}=${request.sessionToken}`);
  if (request.csrfToken) headers.set("X-CSRF-Token", request.csrfToken);

  const response = await fetch(`${API_URL}${path}`, {
    method: request.method ?? "GET",
    headers,
    body: request.body === undefined ? undefined : JSON.stringify(request.body),
    cache: "no-store",
    signal: AbortSignal.timeout(API_TIMEOUT_MS),
  });

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

/** Compte connecté, vu par l'API. Schéma OpenAPI : UserView */
export interface ApiUserView {
  id: string;
  displayName: string;
  role: "citizen" | "hero" | "moderator";
  email: string;
}

const roleFromApi: Record<ApiUserView["role"], UserRole> = {
  citizen: "citoyen",
  hero: "super-heros",
  moderator: "admin",
};

/** Convertit un compte de l'API vers `AuthUser` : seul endroit où les rôles de l'API apparaissent. */
export function fromApiUser(user: ApiUserView): AuthUser {
  const role = roleFromApi[user.role];
  if (!role) throw new Error(`Rôle inconnu renvoyé par l'API : ${user.role}`);
  return { id: user.id, displayName: user.displayName, role };
}
