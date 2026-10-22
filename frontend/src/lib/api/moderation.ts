import { articlesMock } from "@/mocks/articles";
import { citizensMock } from "@/mocks/citizens";
import { heroApplicationsMock } from "@/mocks/hero-applications";
import { incidentsMock } from "@/mocks/incidents";
import { reviewsMock } from "@/mocks/reviews";
import { videosMock } from "@/mocks/videos";
import type { ModerationSummary } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { type ApiSchema, apiFetch } from "./http";
import { userFromToken } from "./session-user";

/**
 * Compteurs du tableau de bord : éléments en attente dans chaque section du back-office.
 * API : `GET /admin/moderation/summary`. Les comptes à contrôler n'y ont pas d'équivalent
 * (`null` : tuile masquée).
 */
export async function getModerationSummary(token: string): Promise<ModerationSummary> {
  if (isApiEnabled) {
    const response = await apiFetch("/admin/moderation/summary", { sessionToken: token });
    const summary = (await response.json()) as ApiSchema<"ModerationSummary">;
    return { ...summary, accounts: null };
  }

  await userFromToken(token, "admin");
  const pending = <T extends { moderation: string }>(items: T[]) =>
    items.filter(({ moderation }) => moderation === "en-attente").length;

  return mockResponse({
    incidents: pending(incidentsMock),
    draftArticles: articlesMock.filter(({ status }) => status === "brouillon").length,
    videos: pending(videosMock),
    // Les avis sont publiés dès leur envoi : seules les réponses modifiées attendent un modérateur.
    comments: reviewsMock.filter(({ reply }) => reply?.status === "en-attente").length,
    accounts: pending(citizensMock),
    emailChanges: citizensMock.filter(({ pendingEmail }) => pendingEmail).length,
    heroApplications: heroApplicationsMock.filter(({ status }) => status === "en-attente").length,
  });
}
