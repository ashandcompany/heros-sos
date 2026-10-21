import { videosMock } from "@/mocks/videos";
import { apiFailureMessage } from "@/lib/validation/form-state";
import type { HeroVideo, ModerationStatus, MutationResult } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { getImdbTitle } from "./imdb";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Vidéothèques des super-héros : seuls les films validés par la modération sont publiés.
 * Un film peut être rattaché à plusieurs héros : un rattachement se désigne par le couple
 * (héros, film), et l'API renvoie une entrée par rattachement.
 */

type ApiFilmStatus = NonNullable<ApiSchema<"Film">["moderation"]>;

const statusFromApi: Record<ApiFilmStatus, ModerationStatus> = {
  pending: "en-attente",
  approved: "valide",
  rejected: "rejete",
};

const statusToApi: Record<ModerationStatus, ApiFilmStatus> = {
  "en-attente": "pending",
  valide: "approved",
  rejete: "rejected",
};

function fromApiFilm(film: ApiSchema<"Film">): HeroVideo {
  if (!film.hero) throw new Error(`Film ${film.id} renvoyé par l'API sans super-héros`);
  return {
    id: film.id,
    hero: film.hero,
    title: film.title,
    description: film.plot,
    videoUrl: film.videoUrl ?? (film.imdbId ? `https://www.imdb.com/title/${film.imdbId}/` : ""),
    imdbId: film.imdbId,
    year: film.releaseYear,
    addedAt: film.addedAt,
    // Les listes publiques ne contiennent que des rattachements approuvés.
    moderation: film.moderation ? statusFromApi[film.moderation] : "valide",
  };
}

/** Tous les films d'une liste paginée de l'API, page après page (100 au plus par page). */
async function fetchAllFilms(
  path: string,
  sessionToken?: string,
  filters: Record<string, string | undefined> = {},
): Promise<HeroVideo[]> {
  const films: HeroVideo[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(
      `${path}${pageQuery({ limit: MAX_PAGE_SIZE, offset }, filters)}`,
      { sessionToken },
    );
    const page = (await response.json()) as ApiSchema<"HeroFilmPage">;
    films.push(...page.items.map(fromApiFilm));
    if (page.items.length === 0 || films.length >= page.total) return films;
  }
}

const byMostRecent = (a: HeroVideo, b: HeroVideo) => Date.parse(b.addedAt) - Date.parse(a.addedAt);

const filmErrorMessages: Record<string, string> = {
  film_not_found: "Film introuvable.",
  film_already_linked: "Ce film est déjà dans votre vidéothèque.",
  film_not_pending: "Ce film a déjà été modéré.",
  not_found: "Film introuvable.",
  catalog_unavailable:
    "Le catalogue IMDb est momentanément indisponible. Réessayez dans quelques instants.",
};

/** Traduit l'échec d'une écriture sur une vidéothèque en message affichable. */
function toFilmMutationFailure(error: unknown, context: string): MutationResult {
  if (error instanceof ApiError && filmErrorMessages[error.code]) {
    return { ok: false, error: filmErrorMessages[error.code] };
  }
  return { ok: false, error: apiFailureMessage(toApiFailure(error, context)) };
}

/* ---------- Back-office (rôle admin) ---------- */

/** Clé d'un rattachement (héros, film), transmise par les formulaires du back-office. */
export function heroVideoKey(video: Pick<HeroVideo, "id" | "hero">): string {
  return `${video.hero.id}/${video.id}`;
}

/** Inverse de `heroVideoKey` ; `null` si la clé est mal formée. */
export function parseHeroVideoKey(key: string): { heroId: string; filmId: string } | null {
  const [heroId, filmId, ...rest] = key.split("/");
  return heroId && filmId && rest.length === 0 ? { heroId, filmId } : null;
}

/**
 * Films ajoutés par les super-héros, des plus récents aux plus anciens.
 * API : `GET /admin/hero-films?status=…`, page après page.
 */
export async function getHeroVideos(
  token: string,
  moderation?: ModerationStatus,
): Promise<HeroVideo[]> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    return mockResponse(
      videosMock
        .filter((video) => !moderation || video.moderation === moderation)
        .sort(byMostRecent),
    );
  }

  const films = await fetchAllFilms("/admin/hero-films", token, {
    status: moderation && statusToApi[moderation],
  });
  return films.sort(byMostRecent);
}

/**
 * Publie ou refuse un film d'un héros (un film refusé peut être republié, et inversement).
 * L'API ne permet pas de remettre un film en attente.
 * API : `POST /admin/hero-films/{heroId}/{filmId}/approve` ou `…/reject`.
 */
export async function moderateVideo(
  token: string,
  heroId: string,
  filmId: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  if (decision === "en-attente") {
    return { ok: false, error: "Un film ne peut pas être remis en attente." };
  }

  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const video = videosMock.find(({ id, hero }) => id === filmId && hero.id === heroId);
    if (!video) return mockResponse({ ok: false, error: "Film introuvable." });

    video.moderation = decision;
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch(
      `/admin/hero-films/${encodeURIComponent(heroId)}/${encodeURIComponent(filmId)}/${decision === "valide" ? "approve" : "reject"}`,
      { method: "POST", sessionToken: token },
    );
    return { ok: true };
  } catch (error) {
    return toFilmMutationFailure(error, "Modération d'un film");
  }
}

/**
 * Retire un film de la vidéothèque d'un héros.
 * API : `DELETE /admin/hero-films/{heroId}/{filmId}`.
 */
export async function deleteVideo(
  token: string,
  heroId: string,
  filmId: string,
): Promise<MutationResult> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const index = videosMock.findIndex(({ id, hero }) => id === filmId && hero.id === heroId);
    if (index === -1) return mockResponse({ ok: false, error: "Film introuvable." });

    videosMock.splice(index, 1);
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch(
      `/admin/hero-films/${encodeURIComponent(heroId)}/${encodeURIComponent(filmId)}`,
      { method: "DELETE", sessionToken: token },
    );
    return { ok: true };
  } catch (error) {
    return toFilmMutationFailure(error, "Suppression d'un film");
  }
}

/* ---------- Espace super-héros (rôle super-heros) ---------- */

/**
 * Films de la vidéothèque du héros connecté, quel que soit leur statut de modération.
 * API : `GET /hero/films`, page après page.
 */
export async function getMyVideos(token: string): Promise<HeroVideo[]> {
  if (!isApiEnabled) {
    const user = await userFromToken(token, "super-heros");
    return mockResponse(videosMock.filter((video) => video.hero.id === user.id).sort(byMostRecent));
  }

  return (await fetchAllFilms("/hero/films", token)).sort(byMostRecent);
}

/**
 * Ajoute un film du catalogue IMDb ; il sera publié après validation par la modération.
 * API : `POST /hero/films` avec `{ imdbId }`.
 */
export async function addMyVideo(token: string, imdbId: string): Promise<MutationResult> {
  if (!isApiEnabled) return addMyVideoWithMocks(token, imdbId);

  try {
    await apiFetch("/hero/films", {
      method: "POST",
      sessionToken: token,
      body: { imdbId } satisfies ApiSchema<"FilmInput">,
    });
    return { ok: true };
  } catch (error) {
    return toFilmMutationFailure(error, "Ajout d'un film");
  }
}

/** API : `DELETE /hero/films/{filmId}`. */
export async function deleteMyVideo(token: string, id: string): Promise<MutationResult> {
  if (!isApiEnabled) {
    const user = await userFromToken(token, "super-heros");
    const index = videosMock.findIndex((video) => video.id === id && video.hero.id === user.id);
    if (index === -1) return mockResponse({ ok: false, error: "Film introuvable." });

    videosMock.splice(index, 1);
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch(`/hero/films/${encodeURIComponent(id)}`, {
      method: "DELETE",
      sessionToken: token,
    });
    return { ok: true };
  } catch (error) {
    return toFilmMutationFailure(error, "Retrait d'un film");
  }
}

/* ---------- Vidéothèque publique ---------- */

/**
 * Films publiés par la modération, des plus récents aux plus anciens, éventuellement ceux
 * d'un seul super-héros (aucun s'il est inconnu de l'API).
 * API : `GET /films` ou `GET /heroes/{heroId}/films`, page après page.
 */
export async function getPublishedVideos(heroId?: string): Promise<HeroVideo[]> {
  if (!isApiEnabled) {
    return mockResponse(
      videosMock
        .filter((video) => video.moderation === "valide")
        .filter((video) => !heroId || video.hero.id === heroId)
        .sort(byMostRecent),
    );
  }

  const path = heroId ? `/heroes/${encodeURIComponent(heroId)}/films` : "/films";
  try {
    return (await fetchAllFilms(path)).sort(byMostRecent);
  } catch (error) {
    // Héros inconnu de l'API (404, ou identifiant mock refusé en 422) : aucun film.
    if (heroId && error instanceof ApiError && (error.status === 404 || error.status === 422)) {
      return [];
    }
    throw error;
  }
}

/* ---------- Mocks ---------- */

async function addMyVideoWithMocks(token: string, imdbId: string): Promise<MutationResult> {
  const user = await userFromToken(token, "super-heros");
  const title = await getImdbTitle(token, imdbId);
  if (!title) return mockResponse({ ok: false, error: "Film introuvable sur IMDb." });
  if (videosMock.some((video) => video.hero.id === user.id && video.imdbId === imdbId)) {
    return mockResponse({ ok: false, error: "Ce film est déjà dans votre vidéothèque." });
  }

  videosMock.push({
    id: `video-${crypto.randomUUID()}`,
    hero: { id: user.id, alias: user.displayName },
    title: title.title,
    description: title.plot,
    videoUrl: `https://www.imdb.com/title/${title.id}/`,
    imdbId: title.id,
    year: title.year,
    addedAt: new Date().toISOString(),
    moderation: "en-attente",
  });
  return mockResponse({ ok: true });
}
