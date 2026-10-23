import { superHeroesMock } from "@/mocks/super-heroes";
import { videosMock } from "@/mocks/videos";
import { villainFilmsMock } from "@/mocks/villain-films";
import { villainsMock } from "@/mocks/villains";
import { apiFailureMessage } from "@/lib/validation/form-state";
import type {
  FilmFilter,
  FilmHero,
  FilmSource,
  HeroVideo,
  ModerationStatus,
  MutationResult,
  Page,
  PageRequest,
  PublicFilm,
  SuperHeroSummary,
  VillainFilm,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { getImdbTitle } from "./imdb";
import { fromApiPage, MAX_PAGE_SIZE, pageQuery, toMockPage } from "./pagination";
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

/** Longueur maximale d'une recherche par titre acceptée par l'API (au-delà : 422). */
export const FILM_QUERY_MAX_LENGTH = 100;

/** Recherche par titre transmise à l'API : `undefined` si vide, tronquée à la longueur permise. */
function filmQuery(query: string | undefined): string | undefined {
  return query?.trim().slice(0, FILM_QUERY_MAX_LENGTH) || undefined;
}

const filmErrorMessages: Record<string, string> = {
  film_not_found: "Film introuvable.",
  film_already_linked: "Ce film est déjà dans votre vidéothèque.",
  film_not_pending: "Ce film a déjà été modéré.",
  not_found: "Film introuvable.",
  catalog_unavailable:
    "Le catalogue IMDb est momentanément indisponible. Réessayez dans quelques instants.",
  validation_failed: "Certaines informations sont invalides. Vérifiez le formulaire.",
};

/** Corps de l'ajout d'un film : identifiant IMDb, ou titre, description et lien d'une vidéo libre. */
const filmInput = (source: FilmSource): ApiSchema<"FilmInput"> =>
  typeof source === "string" ? { imdbId: source } : source;

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
 * Une page des films ajoutés par les super-héros, des plus récents aux plus anciens, filtrés par
 * statut, héros et titre (aucun pour un héros inconnu ou non validé).
 * API : `GET /admin/hero-films?status=…&hero=…&q=…`.
 */
export async function getHeroVideosPage(
  token: string,
  filter: FilmFilter & { moderation?: ModerationStatus } = {},
  page: PageRequest = {},
): Promise<Page<HeroVideo>> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const matches = matchesFilmFilter(filter);
    return mockResponse(
      toMockPage(
        videosMock
          .filter((video) => !filter.moderation || video.moderation === filter.moderation)
          .filter((video) => matches([video.hero], video.title))
          .sort(byMostRecent),
        page,
      ),
    );
  }

  try {
    const response = await apiFetch(
      `/admin/hero-films${pageQuery(page, {
        status: filter.moderation && statusToApi[filter.moderation],
        hero: filter.heroId,
        q: filmQuery(filter.query),
      })}`,
      { sessionToken: token },
    );
    return fromApiPage((await response.json()) as ApiSchema<"HeroFilmPage">, fromApiFilm);
  } catch (error) {
    // Héros inconnu ou non validé (404, ou identifiant mock refusé en 422) : aucun film.
    if (
      filter.heroId &&
      error instanceof ApiError &&
      (error.status === 404 || error.status === 422)
    ) {
      return toMockPage([], page);
    }
    throw error;
  }
}

/**
 * Tous les films d'un super-héros, quel que soit leur statut (pour signaler ceux déjà dans sa
 * vidéothèque) ; aucun s'il est inconnu ou non validé.
 * API : `GET /admin/hero-films?hero=…`, page après page.
 */
export async function getHeroVideos(token: string, heroId: string): Promise<HeroVideo[]> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    return mockResponse(videosMock.filter((video) => video.hero.id === heroId));
  }

  try {
    return await fetchAllFilms("/admin/hero-films", token, { hero: heroId });
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 422)) return [];
    throw error;
  }
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

/**
 * Super-héros auxquels un modérateur peut ajouter un film, par pseudo. Les fiches publiques
 * restant mockées, la liste vient des comptes de l'API (seuls leurs identifiants sont valides).
 * API : `GET /admin/users?role=hero`, page après page.
 */
export async function getHeroesForVideos(token: string): Promise<SuperHeroSummary[]> {
  const byAlias = (a: SuperHeroSummary, b: SuperHeroSummary) =>
    a.alias.localeCompare(b.alias, "fr");

  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    return mockResponse(superHeroesMock.map(({ id, alias }) => ({ id, alias })).sort(byAlias));
  }

  const heroes: SuperHeroSummary[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(
      `/admin/users${pageQuery({ limit: MAX_PAGE_SIZE, offset }, { role: "hero", status: "active" })}`,
      { sessionToken: token },
    );
    const page = (await response.json()) as ApiSchema<"UserSummaryPage">;
    heroes.push(...page.items.map(({ id, alias, email }) => ({ id, alias: alias ?? email })));
    if (page.items.length === 0 || heroes.length >= page.total) return heroes.sort(byAlias);
  }
}

/**
 * Ajoute un film du catalogue IMDb à la vidéothèque d'un héros, directement publié.
 * API : `POST /admin/hero-films` avec `{ heroId, imdbId }`.
 */
export async function addHeroVideo(
  token: string,
  heroId: string,
  source: FilmSource,
): Promise<MutationResult> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const hero = superHeroesMock.find(({ id }) => id === heroId);
    if (!hero) return mockResponse({ ok: false, error: "Film introuvable." });
    return addMockVideo(
      token,
      { id: hero.id, alias: hero.alias },
      source,
      "valide",
      alreadyInHeroLibrary,
    );
  }

  try {
    await apiFetch("/admin/hero-films", {
      method: "POST",
      sessionToken: token,
      body: { heroId, ...filmInput(source) } satisfies ApiSchema<"AdminHeroFilmInput">,
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError && error.code === "film_already_linked") {
      return { ok: false, error: alreadyInHeroLibrary };
    }
    return toFilmMutationFailure(error, "Ajout d'un film par la modération");
  }
}

const alreadyInHeroLibrary = "Ce film est déjà dans la vidéothèque de ce super-héros.";

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
 * Une page de la vidéothèque du héros connecté, des films les plus récents aux plus anciens.
 * API : `GET /hero/films`.
 */
export async function getMyVideosPage(
  token: string,
  page: PageRequest = {},
): Promise<Page<HeroVideo>> {
  if (!isApiEnabled) {
    const user = await userFromToken(token, "super-heros");
    return mockResponse(
      toMockPage(videosMock.filter((video) => video.hero.id === user.id).sort(byMostRecent), page),
    );
  }

  const response = await apiFetch(`/hero/films${pageQuery(page)}`, { sessionToken: token });
  return fromApiPage((await response.json()) as ApiSchema<"HeroFilmPage">, fromApiFilm);
}

/**
 * Ajoute un film du catalogue IMDb ; il sera publié après validation par la modération.
 * API : `POST /hero/films` avec `{ imdbId }`.
 */
export async function addMyVideo(token: string, source: FilmSource): Promise<MutationResult> {
  if (!isApiEnabled) {
    const user = await userFromToken(token, "super-heros");
    return addMockVideo(
      token,
      { id: user.id, alias: user.displayName },
      source,
      "en-attente",
      "Ce film est déjà dans votre vidéothèque.",
    );
  }

  try {
    await apiFetch("/hero/films", {
      method: "POST",
      sessionToken: token,
      body: filmInput(source),
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
 * Une page de la vidéothèque publique : chaque film une seule fois, avec tous ses héros, du
 * rattachement le plus récent au plus ancien. Filtrée par héros et par titre.
 * API : `GET /films?hero=…&q=…`.
 */
export async function getPublishedFilmsPage(
  filter: FilmFilter = {},
  page: PageRequest = {},
): Promise<Page<PublicFilm>> {
  if (!isApiEnabled) {
    const matches = matchesFilmFilter(filter);
    return mockResponse(
      toMockPage(
        publicFilmsFromMocks().filter((film) => matches(film.heroes, film.title)),
        page,
      ),
    );
  }

  try {
    const response = await apiFetch(
      `/films${pageQuery(page, { hero: filter.heroId, q: filmQuery(filter.query) })}`,
    );
    return fromApiPage((await response.json()) as ApiSchema<"PublicFilmPage">, fromApiPublicFilm);
  } catch (error) {
    // Identifiant de héros mal formé (lien ancien ou saisi à la main) : aucun film.
    if (filter.heroId && error instanceof ApiError && error.status === 422) {
      return toMockPage([], page);
    }
    throw error;
  }
}

/**
 * Super-héros ayant au moins un film publié, par pseudo, avec leur nombre de films.
 * API : `GET /films/heroes`.
 */
export async function getFilmHeroes(): Promise<FilmHero[]> {
  if (!isApiEnabled) {
    const heroes = new Map<string, FilmHero>();
    for (const { heroes: filmHeroes } of publicFilmsFromMocks()) {
      for (const hero of filmHeroes) {
        const entry = heroes.get(hero.id) ?? { ...hero, filmCount: 0 };
        entry.filmCount += 1;
        heroes.set(hero.id, entry);
      }
    }
    return mockResponse([...heroes.values()].sort(byHeroAlias));
  }

  const response = await apiFetch("/films/heroes");
  const heroes = (await response.json()) as ApiSchema<"FilmHeroCount">[];
  return heroes.map(({ id, alias, filmCount }) => ({ id, alias, filmCount })).sort(byHeroAlias);
}

/**
 * Films publiés d'un super-héros, des plus récents aux plus anciens (aucun s'il est inconnu de
 * l'API). API : `GET /heroes/{heroId}/films`, page après page.
 */
export async function getPublishedVideos(heroId: string): Promise<HeroVideo[]> {
  if (!isApiEnabled) {
    return mockResponse(
      videosMock
        .filter((video) => video.moderation === "valide" && video.hero.id === heroId)
        .sort(byMostRecent),
    );
  }

  try {
    return (await fetchAllFilms(`/heroes/${encodeURIComponent(heroId)}/films`)).sort(byMostRecent);
  } catch (error) {
    // Héros inconnu de l'API (404, ou identifiant mock refusé en 422) : aucun film.
    if (error instanceof ApiError && (error.status === 404 || error.status === 422)) return [];
    throw error;
  }
}

function fromApiPublicFilm(film: ApiSchema<"PublicFilm">): PublicFilm {
  return {
    id: film.id,
    heroes: film.heroes,
    title: film.title,
    description: film.plot,
    videoUrl: film.videoUrl ?? (film.imdbId ? `https://www.imdb.com/title/${film.imdbId}/` : ""),
    imdbId: film.imdbId,
    year: film.releaseYear,
    addedAt: film.addedAt,
  };
}

const byHeroAlias = (a: SuperHeroSummary, b: SuperHeroSummary) =>
  a.alias.localeCompare(b.alias, "fr");

/* ---------- Films des super-vilains ---------- */

/*
 * Un modérateur rattache un film à la fiche d'un vilain : il est visible aussitôt, sans
 * modération. Comme pour les héros, un rattachement se désigne par le couple (vilain, film).
 */

function fromApiVillainFilm(film: ApiSchema<"Film">): VillainFilm {
  if (!film.villain) throw new Error(`Film ${film.id} renvoyé par l'API sans super-vilain`);
  return {
    id: film.id,
    villain: film.villain,
    title: film.title,
    description: film.plot,
    videoUrl: film.videoUrl ?? (film.imdbId ? `https://www.imdb.com/title/${film.imdbId}/` : ""),
    imdbId: film.imdbId,
    year: film.releaseYear,
    addedAt: film.addedAt,
  };
}

const byMostRecentLink = (a: VillainFilm, b: VillainFilm) =>
  Date.parse(b.addedAt) - Date.parse(a.addedAt);

/**
 * Films rattachés à un super-vilain, des plus récents aux plus anciens (aucun s'il est inconnu
 * de l'API). Public. API : `GET /villains/{villainId}/films`, page après page.
 */
export async function getVillainFilms(villainId: string): Promise<VillainFilm[]> {
  if (!isApiEnabled) {
    return mockResponse(
      villainFilmsMock.filter(({ villain }) => villain.id === villainId).sort(byMostRecentLink),
    );
  }

  const films: VillainFilm[] = [];
  try {
    for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
      const response = await apiFetch(
        `/villains/${encodeURIComponent(villainId)}/films${pageQuery({ limit: MAX_PAGE_SIZE, offset })}`,
      );
      const page = (await response.json()) as ApiSchema<"VillainFilmPage">;
      films.push(...page.items.map(fromApiVillainFilm));
      if (page.items.length === 0 || films.length >= page.total) break;
    }
  } catch (error) {
    // Vilain inconnu de l'API (404, ou identifiant mock refusé en 422) : aucun film.
    if (error instanceof ApiError && (error.status === 404 || error.status === 422)) return [];
    throw error;
  }
  return films.sort(byMostRecentLink);
}

const alreadyInVillainFile = "Ce film est déjà sur la fiche de ce super-vilain.";

/**
 * Rattache un film du catalogue IMDb à la fiche d'un super-vilain, visible aussitôt.
 * API : `POST /admin/villain-films` avec `{ villainId, imdbId }`.
 */
export async function addVillainFilm(
  token: string,
  villainId: string,
  imdbId: string,
): Promise<MutationResult> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const villain = villainsMock.find(({ id }) => id === villainId);
    const title = await getImdbTitle(token, imdbId);
    if (!villain || !title) return mockResponse({ ok: false, error: "Film introuvable." });
    if (villainFilmsMock.some((film) => film.villain.id === villainId && film.imdbId === imdbId)) {
      return mockResponse({ ok: false, error: alreadyInVillainFile });
    }

    villainFilmsMock.push({
      id: `villain-film-${crypto.randomUUID()}`,
      villain: { id: villain.id, alias: villain.alias },
      title: title.title,
      description: title.plot,
      videoUrl: `https://www.imdb.com/title/${title.id}/`,
      imdbId: title.id,
      year: title.year,
      addedAt: new Date().toISOString(),
    });
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch("/admin/villain-films", {
      method: "POST",
      sessionToken: token,
      body: { villainId, imdbId } satisfies ApiSchema<"AdminVillainFilmInput">,
    });
    return { ok: true };
  } catch (error) {
    // Le 409 de cette route n'a pas de corps documenté : on se fie au statut.
    if (error instanceof ApiError && error.status === 409) {
      return { ok: false, error: alreadyInVillainFile };
    }
    return toFilmMutationFailure(error, "Ajout d'un film à un super-vilain");
  }
}

/**
 * Retire un film de la fiche d'un super-vilain.
 * API : `DELETE /admin/villains/{villainId}/films/{filmId}`.
 */
export async function deleteVillainFilm(
  token: string,
  villainId: string,
  filmId: string,
): Promise<MutationResult> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const index = villainFilmsMock.findIndex(
      ({ id, villain }) => id === filmId && villain.id === villainId,
    );
    if (index === -1) return mockResponse({ ok: false, error: "Film introuvable." });

    villainFilmsMock.splice(index, 1);
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch(
      `/admin/villains/${encodeURIComponent(villainId)}/films/${encodeURIComponent(filmId)}`,
      { method: "DELETE", sessionToken: token },
    );
    return { ok: true };
  } catch (error) {
    return toFilmMutationFailure(error, "Retrait d'un film d'un super-vilain");
  }
}

/* ---------- Mocks ---------- */

/** Mock : films publiés regroupés comme le fait l'API (un film IMDb partagé par plusieurs héros). */
function publicFilmsFromMocks(): PublicFilm[] {
  const films = new Map<string, PublicFilm>();
  for (const video of videosMock.filter(({ moderation }) => moderation === "valide")) {
    const key = video.imdbId ?? video.id;
    const film = films.get(key);
    if (!film) {
      const { id, title, description, videoUrl, imdbId, year, addedAt } = video;
      films.set(key, {
        id,
        title,
        description,
        videoUrl,
        imdbId,
        year,
        addedAt,
        heroes: [video.hero],
      });
      continue;
    }
    film.heroes = [...film.heroes, video.hero].sort(byHeroAlias);
    if (Date.parse(video.addedAt) > Date.parse(film.addedAt)) film.addedAt = video.addedAt;
  }
  return [...films.values()].sort((a, b) => Date.parse(b.addedAt) - Date.parse(a.addedAt));
}

/** Mock : filtre d'une liste de films, comme l'API (titre sans tenir compte de la casse). */
function matchesFilmFilter({ heroId, query }: FilmFilter) {
  const needle = filmQuery(query)?.toLocaleLowerCase("fr");
  return (heroes: SuperHeroSummary[], title: string) =>
    (!heroId || heroes.some(({ id }) => id === heroId)) &&
    (!needle || title.toLocaleLowerCase("fr").includes(needle));
}

/** Mock : ajoute un film IMDb (une fois par héros) ou une vidéo libre à la vidéothèque de `hero`. */
async function addMockVideo(
  token: string,
  hero: SuperHeroSummary,
  source: FilmSource,
  moderation: ModerationStatus,
  alreadyOwned: string,
): Promise<MutationResult> {
  const video = {
    id: `video-${crypto.randomUUID()}`,
    hero,
    addedAt: new Date().toISOString(),
    moderation,
  };
  if (typeof source !== "string") {
    videosMock.push({ ...video, ...source });
    return mockResponse({ ok: true });
  }

  const title = await getImdbTitle(token, source);
  if (!title) return mockResponse({ ok: false, error: "Film introuvable sur IMDb." });
  if (videosMock.some(({ hero: { id }, imdbId }) => id === hero.id && imdbId === source)) {
    return mockResponse({ ok: false, error: alreadyOwned });
  }
  videosMock.push({
    ...video,
    title: title.title,
    description: title.plot,
    videoUrl: `https://www.imdb.com/title/${title.id}/`,
    imdbId: title.id,
    year: title.year,
  });
  return mockResponse({ ok: true });
}
