import { articlesMock } from "@/mocks/articles";
import { citizensMock } from "@/mocks/citizens";
import { heroApplicationsMock } from "@/mocks/hero-applications";
import { incidentsMock } from "@/mocks/incidents";
import { newslettersMock } from "@/mocks/newsletters";
import { reviewsMock } from "@/mocks/reviews";
import { videosMock } from "@/mocks/videos";
import type { ModerationSummary } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { type ApiSchema, apiFetch } from "./http";
import { userFromToken } from "./session-user";

/**
 * Compteurs du tableau de bord : éléments en attente dans chaque section du back-office.
 * API : `GET /admin/moderation/summary`.
 */
export async function getModerationSummary(token: string): Promise<ModerationSummary> {
  if (isApiEnabled) {
    const response = await apiFetch("/admin/moderation/summary", { sessionToken: token });
    return (await response.json()) as ApiSchema<"ModerationSummary">;
  }

  await userFromToken(token, "admin");
  const pending = <T extends { moderation: string }>(items: T[]) =>
    items.filter(({ moderation }) => moderation === "en-attente").length;

  return mockResponse({
    incidents: pending(incidentsMock),
    draftArticles: articlesMock.filter(({ status }) => status === "brouillon").length,
    draftNewsletters: newslettersMock.filter(({ sentAt }) => !sentAt).length,
    videos: pending(videosMock),
    // Les avis sont publiés dès leur envoi : seules les réponses modifiées attendent un modérateur.
    comments: reviewsMock.filter(({ reply }) => reply?.status === "en-attente").length,
    emailChanges: citizensMock.filter(({ pendingEmail }) => pendingEmail).length,
    heroApplications: heroApplicationsMock.filter(({ status }) => status === "en-attente").length,
  });
}
