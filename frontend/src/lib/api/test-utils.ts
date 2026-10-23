import { vi } from "vitest";

/*
 * Tests de `lib/api/` avec l'API activée : à importer en tête du fichier de test, avant le module
 * testé. Les fonctions de conversion (`fromApi*`) restent privées : on les vérifie à travers les
 * fonctions publiques, en simulant les réponses de l'API.
 *
 * `http.ts` importe `next/headers` et `next/navigation`, qui n'existent qu'à l'intérieur de Next.
 */
vi.mock("next/headers", () => ({ headers: async () => new Headers() }));
vi.mock("next/navigation", () => ({
  redirect: vi.fn(),
  // Hors de Next, aucune erreur n'est une redirection à relancer.
  unstable_rethrow: () => {},
}));

export const TEST_API_URL = "http://api.test/api/v1";

/**
 * Active l'API et renvoie le module demandé, rechargé : `config.ts` lit `API_URL` au chargement,
 * et `cache()` de React ne doit pas garder de valeur d'un test à l'autre.
 *
 * `const { getVillain } = await withApi(() => import("./villains"));`
 */
export async function withApi<Module>(load: () => Promise<Module>): Promise<Module> {
  vi.stubEnv("API_URL", TEST_API_URL);
  vi.resetModules();
  return load();
}

/** Réponse JSON de l'API, avec l'enveloppe d'erreur `{ error: { code } }` si `status` n'est pas 2xx. */
export function jsonResponse(body: unknown, init: ResponseInit = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    ...init,
    headers: { "Content-Type": "application/json", ...init.headers },
  });
}

export const errorResponse = (status: number, code: string, headers?: HeadersInit) =>
  jsonResponse({ error: { code, message: code } }, { status, headers });

/**
 * Remplace `fetch` : chaque appel reçoit la réponse de `route` pour son chemin (sans `API_URL`).
 * Renvoie le mock pour vérifier les appels.
 */
export function mockFetch(route: (path: string) => Response) {
  const fetchMock = vi.fn(async (input: string | URL | Request) => {
    const url = typeof input === "string" ? input : input instanceof URL ? input.href : input.url;
    return route(url.replace(TEST_API_URL, ""));
  });
  vi.stubGlobal("fetch", fetchMock);
  return fetchMock;
}
