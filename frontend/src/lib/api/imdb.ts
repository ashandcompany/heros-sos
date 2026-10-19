import { imdbCatalogMock } from "@/mocks/imdb";
import type { ImdbTitle } from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

/*
 * Catalogue IMDb, relayé par l'API Go : la clé d'accès à IMDb reste côté serveur
 * et n'est jamais exposée au navigateur.
 */

const normalize = (value: string) =>
  value
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase();

/**
 * Films dont le titre contient la recherche (10 au maximum).
 * Future implémentation : `GET ${API_URL}/imdb/titles?q=${query}`
 */
export async function searchImdbTitles(token: string, query: string): Promise<ImdbTitle[]> {
  await userFromToken(token, "super-heros");
  const wanted = normalize(query.trim());
  if (!wanted) return mockResponse([]);
  return mockResponse(
    imdbCatalogMock.filter(({ title }) => normalize(title).includes(wanted)).slice(0, 10),
  );
}

/** Future implémentation : `GET ${API_URL}/imdb/titles/${id}` (404 → `null`). */
export async function getImdbTitle(id: string): Promise<ImdbTitle | null> {
  return mockResponse(imdbCatalogMock.find((title) => title.id === id) ?? null);
}
