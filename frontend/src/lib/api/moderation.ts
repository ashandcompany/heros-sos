import { articlesMock } from "@/mocks/articles";
import { citizensMock } from "@/mocks/citizens";
import { heroApplicationsMock } from "@/mocks/hero-applications";
import { incidentsMock } from "@/mocks/incidents";
import { reviewsMock } from "@/mocks/reviews";
import { videosMock } from "@/mocks/videos";
import type { ModerationSummary } from "@/types";
import { countDraftArticles } from "./blog";
import { isApiEnabled, mockResponse } from "./config";
import { getHeroApplications } from "./hero-requests";
import { apiFetch } from "./http";
import { pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/** Nombre d'éléments `pending` d'une liste paginée du back-office (champ `total`). */
async function countPending(path: string, token: string): Promise<number> {
  const response = await apiFetch(`${path}${pageQuery({ limit: 1 }, { status: "pending" })}`, {
    sessionToken: token,
  });
  return ((await response.json()) as { total: number }).total;
}

/**
 * Compteurs du tableau de bord : éléments en attente dans chaque section du back-office.
 * Avec l'API, les demandes de super-héros (`GET /admin/hero-requests`), les films
 * (`GET /admin/hero-films`), les réponses des héros à vérifier (`GET /admin/hero-review-replies`)
 * les changements d'email (`GET /admin/email-change-requests`) et les brouillons du blog
 * (`GET /admin/articles?status=brouillon`) sont comptés par le `total` de leur liste ; les comptes à contrôler n'ont pas d'équivalent (`null` : tuile masquée) ; les
 * autres compteurs restent mockés.
 * Future implémentation : `GET ${API_URL}/admin/moderation/summary`
 */
export async function getModerationSummary(token: string): Promise<ModerationSummary> {
  await userFromToken(token, "admin");
  const pending = <T extends { moderation: string }>(items: T[]) =>
    items.filter(({ moderation }) => moderation === "en-attente").length;

  const mocked = { incidents: pending(incidentsMock) };

  if (isApiEnabled) {
    const [applications, videos, comments, emailChanges, draftArticles] = await Promise.all([
      getHeroApplications(token, { status: "en-attente", limit: 1 }),
      countPending("/admin/hero-films", token),
      countPending("/admin/hero-review-replies", token),
      countPending("/admin/email-change-requests", token),
      countDraftArticles(token),
    ]);
    return {
      ...mocked,
      draftArticles,
      heroApplications: applications.total,
      videos,
      comments,
      accounts: null,
      emailChanges,
    };
  }

  return mockResponse({
    ...mocked,
    draftArticles: articlesMock.filter(({ status }) => status === "brouillon").length,
    videos: pending(videosMock),
    // Les avis sont publiés dès leur envoi : seules les réponses modifiées attendent un modérateur.
    comments: reviewsMock.filter(({ reply }) => reply?.status === "en-attente").length,
    accounts: pending(citizensMock),
    emailChanges: citizensMock.filter(({ pendingEmail }) => pendingEmail).length,
    heroApplications: heroApplicationsMock.filter(({ status }) => status === "en-attente").length,
  });
}
