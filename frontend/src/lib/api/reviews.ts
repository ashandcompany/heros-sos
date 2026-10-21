import { unstable_rethrow } from "next/navigation";
import { incidentsMock } from "@/mocks/incidents";
import { mockInterventionId, reviewsMock } from "@/mocks/reviews";
import { apiFailureMessage } from "@/lib/validation/form-state";
import type {
  HeroReputation,
  HeroReview,
  Incident,
  ModerationStatus,
  MutationResult,
  RateableIntervention,
  ReplyForModeration,
  ResolvedIncident,
  ReviewReply,
  ReviewStatus,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Avis sur les super-héros. Un avis est publié dès son envoi ; la modération peut le masquer
 * (il ne compte alors plus dans la moyenne). La réponse du héros est publiée à sa création et
 * revérifiée par la modération à chaque modification.
 */

type ApiReviewStatus = ApiSchema<"HeroReview">["status"];
type ApiReplyStatus = ApiSchema<"HeroReviewReply">["status"];

const reviewStatusFromApi: Record<ApiReviewStatus, ReviewStatus> = {
  published: "valide",
  hidden: "rejete",
};

const reviewStatusToApi: Record<ReviewStatus, ApiReviewStatus> = {
  valide: "published",
  rejete: "hidden",
};

const replyStatusFromApi: Record<ApiReplyStatus, ModerationStatus> = {
  published: "valide",
  pending: "en-attente",
  hidden: "rejete",
};

const replyStatusToApi: Record<ModerationStatus, ApiReplyStatus> = {
  valide: "published",
  "en-attente": "pending",
  rejete: "hidden",
};

function fromApiReply(reply: ApiSchema<"HeroReviewReply">): ReviewReply {
  return {
    content: reply.content,
    status: replyStatusFromApi[reply.status],
    createdAt: reply.createdAt,
    editedAt: reply.editedAt ?? null,
  };
}

function fromApiReview(review: ApiSchema<"HeroReview">): HeroReview {
  return {
    id: review.id,
    interventionId: review.interventionId,
    hero: { id: review.heroId, alias: review.heroAlias ?? "" },
    rating: review.rating,
    comment: review.comment ?? null,
    status: reviewStatusFromApi[review.status],
    createdAt: review.createdAt,
    reply: review.reply ? fromApiReply(review.reply) : null,
  };
}

/** Éléments de toutes les pages d'une liste de l'API (100 au plus par appel). */
async function fetchAllPages<Item>(
  path: string,
  sessionToken: string | undefined,
  readPage: (body: unknown) => { items: Item[]; total?: number },
  filters: Record<string, string | undefined> = {},
): Promise<Item[]> {
  const items: Item[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(
      `${path}${pageQuery({ limit: MAX_PAGE_SIZE, offset }, filters)}`,
      { sessionToken },
    );
    const page = readPage(await response.json());
    items.push(...page.items);
    // Sans `total` (avis publics), une page incomplète est la dernière.
    const done =
      page.total === undefined ? page.items.length < MAX_PAGE_SIZE : items.length >= page.total;
    if (page.items.length === 0 || done) return items;
  }
}

const byMostRecent = (a: { createdAt: string }, b: { createdAt: string }) =>
  Date.parse(b.createdAt) - Date.parse(a.createdAt);

const reviewErrorMessages: Record<string, string> = {
  hero_already_reviewed: "Vous avez déjà noté ce super-héros pour cette intervention.",
  hero_review_not_eligible: "Cette action n'est pas possible sur cet avis.",
  hero_review_reply_unavailable: "Ce commentaire n'accepte pas de réponse.",
  not_found: "Avis ou intervention introuvable.",
  validation_failed: "Certaines informations sont invalides.",
};

/** Traduit l'échec d'une écriture en message affichable (codes métier d'abord, 403 compris). */
function toReviewMutationFailure(error: unknown, context: string): MutationResult {
  if (error instanceof ApiError && reviewErrorMessages[error.code]) {
    return { ok: false, error: reviewErrorMessages[error.code] };
  }
  return { ok: false, error: apiFailureMessage(toApiFailure(error, context)) };
}

/** Note moyenne et nombre d'avis publiés (les avis masqués ne comptent pas). */
export function summarizeRatings(reviews: Pick<HeroReview, "rating" | "status">[]): {
  averageRating: number;
  ratingsCount: number;
} {
  const published = reviews.filter(({ status }) => status === "valide");
  const total = published.reduce((sum, { rating }) => sum + rating, 0);
  return {
    averageRating: published.length > 0 ? total / published.length : 0,
    ratingsCount: published.length,
  };
}

/* ---------- Espace citoyen ---------- */

const outcomeFromApi = { succeeded: "reussi", failed: "echoue" } as const;

/** Type et ville d'un incident (`null` s'il ne peut pas être lu : il reste notable). */
async function readIncidentLabel(token: string, incidentId: string): Promise<string | null> {
  try {
    const response = await apiFetch(`/incidents/${encodeURIComponent(incidentId)}`, {
      sessionToken: token,
    });
    const incident = (await response.json()) as ApiSchema<"Incident">;
    return [incident.typeLabel, incident.location.city].filter(Boolean).join(" · ") || null;
  } catch (error) {
    unstable_rethrow(error);
    console.warn(`Avis : incident ${incidentId} illisible`, error);
    return null;
  }
}

/**
 * Super-héros à noter sur les interventions terminées des incidents déclarés par l'utilisateur,
 * des plus récentes aux plus anciennes, avec l'avis déjà donné le cas échéant.
 * API : `GET /me/interventions`, page après page, puis `GET /incidents/{id}` pour le libellé
 * de chaque incident (l'API ne le renvoie pas avec l'intervention).
 */
export async function getMyInterventions(token: string): Promise<RateableIntervention[]> {
  if (!isApiEnabled) return getMyInterventionsWithMocks(token);

  const rows = await fetchAllPages(
    "/me/interventions",
    token,
    (body) => body as ApiSchema<"RateableInterventionPage">,
  );
  const incidentIds = [...new Set(rows.map(({ incidentId }) => incidentId))];
  const labels = new Map(
    await Promise.all(
      incidentIds.map(async (id) => [id, await readIncidentLabel(token, id)] as const),
    ),
  );

  return rows
    .map((row) => ({
      interventionId: row.interventionId,
      incident: { id: row.incidentId, label: labels.get(row.incidentId) ?? null },
      outcome: outcomeFromApi[row.outcome],
      endedAt: row.endedAt,
      hero: { id: row.heroId, alias: row.heroAlias },
      review: row.review ? fromApiReview(row.review) : null,
    }))
    .sort((a, b) => Date.parse(b.endedAt) - Date.parse(a.endedAt));
}

/**
 * Note un super-héros d'une intervention (une seule fois par héros). L'avis est publié aussitôt.
 * API : `POST /interventions/{interventionId}/reviews` avec `{ heroId, rating, comment }`.
 */
export async function rateIntervention(
  token: string,
  interventionId: string,
  heroId: string,
  { rating, comment }: { rating: number; comment: string | null },
): Promise<MutationResult> {
  if (!isApiEnabled) {
    return rateInterventionWithMocks(token, interventionId, heroId, { rating, comment });
  }

  try {
    await apiFetch(`/interventions/${encodeURIComponent(interventionId)}/reviews`, {
      method: "POST",
      sessionToken: token,
      body: {
        heroId,
        rating,
        ...(comment ? { comment } : {}),
      } satisfies ApiSchema<"HeroReviewPayload">,
    });
    return { ok: true };
  } catch (error) {
    return toReviewMutationFailure(error, "Avis sur une intervention");
  }
}

/* ---------- Espace super-héros ---------- */

/**
 * Note globale et avis reçus par le héros connecté. Le commentaire d'un avis masqué n'est pas
 * transmis.
 * API : `GET /hero/reviews`, page après page.
 */
export async function getMyReputation(token: string): Promise<HeroReputation> {
  if (!isApiEnabled) {
    const user = await userFromToken(token, "super-heros");
    const mine = reviewsMock.filter(({ hero }) => hero.id === user.id);
    return mockResponse({
      ...summarizeRatings(mine),
      reviews: [...mine]
        .sort(byMostRecent)
        .map((review) => (review.status === "rejete" ? { ...review, comment: null } : review)),
    });
  }

  let summary = { averageRating: 0, ratingsCount: 0 };
  const reviews = await fetchAllPages("/hero/reviews", token, (body) => {
    const page = body as ApiSchema<"HeroReputation">;
    summary = { averageRating: page.averageRating, ratingsCount: page.ratingsCount };
    return { items: page.ratings, total: page.total };
  });
  return { ...summary, reviews: reviews.map(fromApiReview).sort(byMostRecent) };
}

/**
 * Répond au commentaire publié d'un avis, ou modifie la réponse (elle repasse alors en
 * vérification).
 * API : `PUT /hero/reviews/{reviewId}/reply` avec `{ content }`.
 */
export async function replyToReview(
  token: string,
  reviewId: string,
  content: string,
): Promise<MutationResult> {
  if (!isApiEnabled) {
    const user = await userFromToken(token, "super-heros");
    const review = reviewsMock.find(({ id, hero }) => id === reviewId && hero.id === user.id);
    if (!review?.comment || review.status !== "valide") {
      return mockResponse({ ok: false, error: reviewErrorMessages.hero_review_reply_unavailable });
    }

    const now = new Date().toISOString();
    review.reply = review.reply
      ? { ...review.reply, content, status: "en-attente", editedAt: now }
      : { content, status: "valide", createdAt: now, editedAt: null };
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch(`/hero/reviews/${encodeURIComponent(reviewId)}/reply`, {
      method: "PUT",
      sessionToken: token,
      body: { content } satisfies ApiSchema<"HeroReviewReplyPayload">,
    });
    return { ok: true };
  } catch (error) {
    return toReviewMutationFailure(error, "Réponse à un avis");
  }
}

/* ---------- Fiches publiques de l'encyclopédie ---------- */

/**
 * Avis publiés avec un commentaire, des plus récents aux plus anciens, et les réponses publiées.
 * Un héros inconnu de l'API (404, ou identifiant mock refusé en 422) n'a aucun avis.
 * API : `GET /heroes/{heroId}/reviews`, page après page.
 */
export async function getHeroPublicReviews(heroId: string): Promise<HeroReview[]> {
  if (!isApiEnabled) {
    return mockResponse(
      reviewsMock
        .filter(({ hero, status, comment }) => hero.id === heroId && status === "valide" && comment)
        .map((review) => ({
          ...review,
          reply: review.reply?.status === "valide" ? review.reply : null,
        }))
        .sort(byMostRecent),
    );
  }

  try {
    const reviews = await fetchAllPages(
      `/heroes/${encodeURIComponent(heroId)}/reviews`,
      undefined,
      (body) => ({ items: (body as ApiSchema<"PublicHeroReputation">).reviews }),
    );
    return reviews
      .map(fromApiReview)
      .filter(({ comment }) => comment)
      .sort(byMostRecent);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 422)) return [];
    throw error;
  }
}

/* ---------- Back-office (rôle admin) ---------- */

/**
 * Avis accompagnés d'un commentaire, des plus récents aux plus anciens.
 * API : `GET /admin/hero-reviews?status=…`, page après page.
 */
export async function getReviewsForModeration(
  token: string,
  status?: ReviewStatus,
): Promise<HeroReview[]> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    return mockResponse(
      reviewsMock
        .filter((review) => review.comment && (!status || review.status === status))
        .sort(byMostRecent),
    );
  }

  const reviews = await fetchAllPages(
    "/admin/hero-reviews",
    token,
    (body) => body as ApiSchema<"HeroReviewPage">,
    { status: status && reviewStatusToApi[status] },
  );
  return reviews
    .map(fromApiReview)
    .filter(({ comment }) => comment)
    .sort(byMostRecent);
}

/**
 * Publie ou masque un avis (un avis masqué ne compte plus dans la moyenne du héros).
 * API : `PATCH /admin/hero-reviews/{reviewId}` avec `{ status }`.
 */
export async function moderateReview(
  token: string,
  reviewId: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  if (decision === "en-attente") {
    return { ok: false, error: "Un avis ne peut pas être remis en attente." };
  }

  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const review = reviewsMock.find(({ id }) => id === reviewId);
    if (!review) return mockResponse({ ok: false, error: reviewErrorMessages.not_found });

    review.status = decision;
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch(`/admin/hero-reviews/${encodeURIComponent(reviewId)}`, {
      method: "PATCH",
      sessionToken: token,
      body: { status: reviewStatusToApi[decision] } satisfies ApiSchema<"HeroReviewStatusPayload">,
    });
    return { ok: true };
  } catch (error) {
    return toReviewMutationFailure(error, "Modération d'un avis");
  }
}

/**
 * Réponses des super-héros, des plus récentes aux plus anciennes.
 * API : `GET /admin/hero-review-replies?status=…`, page après page.
 */
export async function getRepliesForModeration(
  token: string,
  status?: ModerationStatus,
): Promise<ReplyForModeration[]> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    return mockResponse(
      reviewsMock
        .flatMap(({ id, hero, rating, comment, status: reviewStatus, reply }) =>
          reply && (!status || reply.status === status)
            ? [{ reviewId: id, heroAlias: hero.alias, rating, comment, reviewStatus, reply }]
            : [],
        )
        .sort((a, b) => byMostRecent(a.reply, b.reply)),
    );
  }

  const items = await fetchAllPages(
    "/admin/hero-review-replies",
    token,
    (body) => body as ApiSchema<"HeroReviewReplyModerationPage">,
    { status: status && replyStatusToApi[status] },
  );
  return items
    .map((item) => ({
      reviewId: item.reply.reviewId,
      heroAlias: item.heroAlias,
      rating: item.rating,
      comment: item.comment ?? null,
      reviewStatus: reviewStatusFromApi[item.reviewStatus],
      reply: fromApiReply(item.reply),
    }))
    .sort((a, b) => byMostRecent(a.reply, b.reply));
}

/**
 * Publie ou masque la réponse d'un super-héros.
 * API : `PATCH /admin/hero-review-replies/{reviewId}` avec `{ status }`.
 */
export async function moderateReply(
  token: string,
  reviewId: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  if (decision === "en-attente") {
    return { ok: false, error: "Une réponse ne peut pas être remise en attente." };
  }

  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const review = reviewsMock.find(({ id }) => id === reviewId);
    if (!review?.reply) return mockResponse({ ok: false, error: "Réponse introuvable." });

    review.reply.status = decision;
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch(`/admin/hero-review-replies/${encodeURIComponent(reviewId)}`, {
      method: "PATCH",
      sessionToken: token,
      body: { status: reviewStatusToApi[decision] } satisfies ApiSchema<"HeroReviewStatusPayload">,
    });
    return { ok: true };
  } catch (error) {
    return toReviewMutationFailure(error, "Modération d'une réponse");
  }
}

/* ---------- Mocks ---------- */

const isResolved = (incident: Incident): incident is ResolvedIncident =>
  incident.status === "resolu" && !!incident.resolvedAt && !!incident.hero;

/** Mock : un incident résolu = une intervention réussie de son héros. */
async function getMyInterventionsWithMocks(token: string): Promise<RateableIntervention[]> {
  const user = await userFromToken(token, ["citoyen", "super-heros"]);
  return mockResponse(
    incidentsMock
      .filter((incident) => incident.reporterId === user.id)
      .filter(isResolved)
      // Un héros déclarant ne peut pas se noter lui-même.
      .filter((incident) => incident.hero.id !== user.id)
      .sort((a, b) => Date.parse(b.resolvedAt) - Date.parse(a.resolvedAt))
      .map((incident) => {
        const interventionId = mockInterventionId(incident.id);
        return {
          interventionId,
          incident: { id: incident.id, label: incident.title },
          outcome: "reussi" as const,
          endedAt: incident.resolvedAt,
          hero: incident.hero,
          review:
            reviewsMock.find(
              (review) =>
                review.interventionId === interventionId && review.hero.id === incident.hero.id,
            ) ?? null,
        };
      }),
  );
}

async function rateInterventionWithMocks(
  token: string,
  interventionId: string,
  heroId: string,
  { rating, comment }: { rating: number; comment: string | null },
): Promise<MutationResult> {
  const user = await userFromToken(token, ["citoyen", "super-heros"]);
  const incident = incidentsMock.find(
    (existing) =>
      mockInterventionId(existing.id) === interventionId &&
      existing.reporterId === user.id &&
      isResolved(existing) &&
      existing.hero?.id === heroId &&
      heroId !== user.id,
  );
  if (!incident?.hero) {
    return mockResponse({ ok: false, error: "Cette intervention ne peut pas être notée." });
  }
  if (
    reviewsMock.some(
      (review) => review.interventionId === interventionId && review.hero.id === heroId,
    )
  ) {
    return mockResponse({ ok: false, error: reviewErrorMessages.hero_already_reviewed });
  }

  reviewsMock.push({
    id: `review-${crypto.randomUUID()}`,
    incidentId: incident.id,
    interventionId,
    hero: incident.hero,
    rating,
    comment,
    status: "valide",
    createdAt: new Date().toISOString(),
    reply: null,
  });
  return mockResponse({ ok: true });
}
