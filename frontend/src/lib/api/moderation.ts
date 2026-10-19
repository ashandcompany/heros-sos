import { articlesMock } from "@/mocks/articles";
import { citizensMock } from "@/mocks/citizens";
import { heroApplicationsMock } from "@/mocks/hero-applications";
import { incidentsMock } from "@/mocks/incidents";
import { ratingsMock } from "@/mocks/ratings";
import { videosMock } from "@/mocks/videos";
import type { ModerationSummary } from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

/**
 * Compteurs du tableau de bord : éléments en attente dans chaque section du back-office.
 * Future implémentation : `GET ${API_URL}/admin/moderation/summary`
 */
export async function getModerationSummary(token: string): Promise<ModerationSummary> {
  await userFromToken(token, "admin");
  const pending = <T extends { moderation: string }>(items: T[]) =>
    items.filter(({ moderation }) => moderation === "en-attente").length;

  return mockResponse({
    incidents: pending(incidentsMock),
    accounts: pending(citizensMock),
    emailChanges: citizensMock.filter(({ pendingEmail }) => pendingEmail).length,
    heroApplications: heroApplicationsMock.filter(({ status }) => status === "en-attente").length,
    videos: pending(videosMock),
    comments: ratingsMock.filter(
      ({ comment, commentModeration }) => comment && commentModeration === "en-attente",
    ).length,
    draftArticles: articlesMock.filter(({ status }) => status === "brouillon").length,
  });
}
