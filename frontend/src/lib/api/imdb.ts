import { imdbCatalogMock } from "@/mocks/imdb";
import type { ImdbSearchResult, ImdbTitle } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { userFromToken } from "./session-user";

/*
 * Catalogue IMDb, relayé par l'API Go (OMDb) : la clé d'accès reste côté serveur et n'est jamais
 * exposée au navigateur.
 */

/** Longueur maximale d'une recherche acceptée par l'API (au-delà : 422). */
const QUERY_MAX_LENGTH = 100;

const normalize = (value: string) =>
  value
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase();

function fromApiTitle(title: ApiSchema<"FilmTitle">): ImdbTitle {
  return {
    id: title.id,
    title: title.title,
    year: title.year,
    plot: title.plot ?? "",
    posterUrl: title.posterUrl,
  };
}

/**
 * Films dont le titre correspond à la recherche (10 au maximum).
 * API : `GET /imdb/titles?q=…` (super-héros et modérateurs).
 */
export async function searchImdbTitles(token: string, query: string): Promise<ImdbSearchResult> {
  const wanted = query.trim().slice(0, QUERY_MAX_LENGTH);

  if (!isApiEnabled) {
    await userFromToken(token, ["super-heros", "admin"]);
    const normalized = normalize(wanted);
    const titles = normalized
      ? imdbCatalogMock.filter(({ title }) => normalize(title).includes(normalized)).slice(0, 10)
      : [];
    return mockResponse({ ok: true, titles });
  }

  if (!wanted) return { ok: true, titles: [] };
  try {
    const response = await apiFetch(`/imdb/titles?${new URLSearchParams({ q: wanted })}`, {
      sessionToken: token,
    });
    const titles = (await response.json()) as ApiSchema<"FilmTitle">[];
    return { ok: true, titles: titles.map(fromApiTitle) };
  } catch (error) {
    if (error instanceof ApiError && error.code === "catalog_unavailable") {
      return { ok: false, error: "catalog-unavailable" };
    }
    return toApiFailure(error, "Recherche dans le catalogue IMDb");
  }
}

/**
 * Fiche d'un film (`null` s'il est absent du catalogue).
 * API : `GET /imdb/titles/{id}` (404 `film_not_found` → `null`).
 */
export async function getImdbTitle(token: string, id: string): Promise<ImdbTitle | null> {
  if (!isApiEnabled) {
    await userFromToken(token, ["super-heros", "admin"]);
    return mockResponse(imdbCatalogMock.find((title) => title.id === id) ?? null);
  }

  try {
    const response = await apiFetch(`/imdb/titles/${encodeURIComponent(id)}`, {
      sessionToken: token,
    });
    return fromApiTitle((await response.json()) as ApiSchema<"FilmTitle">);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}
