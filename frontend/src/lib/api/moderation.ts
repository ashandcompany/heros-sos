import { articlesMock } from "@/mocks/articles";
import { citizensMock } from "@/mocks/citizens";
import { heroApplicationsMock } from "@/mocks/hero-applications";
import { incidentsMock } from "@/mocks/incidents";
import { ratingsMock } from "@/mocks/ratings";
import { videosMock } from "@/mocks/videos";
import type { ModerationSummary } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { getHeroApplications } from "./hero-requests";
import { userFromToken } from "./session-user";

/**
 * Compteurs du tableau de bord : éléments en attente dans chaque section du back-office.
 * Avec l'API, les demandes de super-héros sont comptées par `GET /admin/hero-requests` (`total`) ;
 * les comptes à contrôler et les changements d'email n'ont pas d'équivalent (`null` : tuile
 * masquée) ; les autres compteurs restent mockés.
 * Future implémentation : `GET ${API_URL}/admin/moderation/summary`
 */
export async function getModerationSummary(token: string): Promise<ModerationSummary> {
  await userFromToken(token, "admin");
  const pending = <T extends { moderation: string }>(items: T[]) =>
    items.filter(({ moderation }) => moderation === "en-attente").length;

  const mocked = {
    incidents: pending(incidentsMock),
    videos: pending(videosMock),
    comments: ratingsMock.filter(
      ({ comment, commentModeration }) => comment && commentModeration === "en-attente",
    ).length,
    draftArticles: articlesMock.filter(({ status }) => status === "brouillon").length,
  };

  if (isApiEnabled) {
    const applications = await getHeroApplications(token, { status: "en-attente", limit: 1 });
    return { ...mocked, heroApplications: applications.total, accounts: null, emailChanges: null };
  }

  return mockResponse({
    ...mocked,
    accounts: pending(citizensMock),
    emailChanges: citizensMock.filter(({ pendingEmail }) => pendingEmail).length,
    heroApplications: heroApplicationsMock.filter(({ status }) => status === "en-attente").length,
  });
}
