import { videosMock } from "@/mocks/videos";
import type { HeroVideo, ModerationStatus, MutationResult } from "@/types";
import { mockResponse } from "./config";
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
