import { videosMock } from "@/mocks/videos";
import type { HeroVideo, ModerationStatus, MutationResult } from "@/types";
import { mockResponse } from "./config";
import { getImdbTitle } from "./imdb";
import { userFromToken } from "./session-user";

/* Vidéothèques des super-héros : seuls les films validés par la modération seront publiés. */

/**
 * Films ajoutés par les super-héros, des plus récents aux plus anciens (back-office).
 * Future implémentation : `GET ${API_URL}/admin/videos?moderation=${moderation}`
 */
export async function getHeroVideos(
  token: string,
  moderation?: ModerationStatus,
): Promise<HeroVideo[]> {
  await userFromToken(token, "admin");
  return mockResponse(
    videosMock
      .filter((video) => !moderation || video.moderation === moderation)
      .sort((a, b) => Date.parse(b.addedAt) - Date.parse(a.addedAt)),
  );
}

/** Future implémentation : `PUT ${API_URL}/admin/videos/${id}/moderation` avec `{ decision }`. */
export async function moderateVideo(
  token: string,
  id: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const video = videosMock.find((existing) => existing.id === id);
  if (!video) return mockResponse({ ok: false, error: "Film introuvable." });

  video.moderation = decision;
  return mockResponse({ ok: true });
}

/** Future implémentation : `DELETE ${API_URL}/admin/videos/${id}` */
export async function deleteVideo(token: string, id: string): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const index = videosMock.findIndex((existing) => existing.id === id);
  if (index === -1) return mockResponse({ ok: false, error: "Film introuvable." });

  videosMock.splice(index, 1);
  return mockResponse({ ok: true });
}

/* ---------- Espace super-héros (rôle super-heros) ---------- */

/**
 * Films de la vidéothèque du héros connecté, quel que soit leur statut de modération.
 * Future implémentation : `GET ${API_URL}/hero/videos`
 */
export async function getMyVideos(token: string): Promise<HeroVideo[]> {
  const user = await userFromToken(token, "super-heros");
  return mockResponse(
    videosMock
      .filter((video) => video.hero.id === user.id)
      .sort((a, b) => Date.parse(b.addedAt) - Date.parse(a.addedAt)),
  );
}

/**
 * Ajoute un film du catalogue IMDb ; il sera publié après validation par la modération.
 * Future implémentation : `POST ${API_URL}/hero/videos` avec `{ imdbId }`.
 */
export async function addMyVideo(token: string, imdbId: string): Promise<MutationResult> {
  const user = await userFromToken(token, "super-heros");
  const title = await getImdbTitle(imdbId);
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

/** Future implémentation : `DELETE ${API_URL}/hero/videos/${id}` */
export async function deleteMyVideo(token: string, id: string): Promise<MutationResult> {
  const user = await userFromToken(token, "super-heros");
  const index = videosMock.findIndex((video) => video.id === id && video.hero.id === user.id);
  if (index === -1) return mockResponse({ ok: false, error: "Film introuvable." });

  videosMock.splice(index, 1);
  return mockResponse({ ok: true });
}

/* ---------- Vidéothèque publique ---------- */

/**
 * Films publiés par la modération, des plus récents aux plus anciens, éventuellement ceux
 * d'un seul super-héros.
 * Future implémentation : `GET ${API_URL}/videos?hero=${heroId}`
 */
export async function getPublishedVideos(heroId?: string): Promise<HeroVideo[]> {
  return mockResponse(
    videosMock
      .filter((video) => video.moderation === "valide")
      .filter((video) => !heroId || video.hero.id === heroId)
      .sort((a, b) => Date.parse(b.addedAt) - Date.parse(a.addedAt)),
  );
}
