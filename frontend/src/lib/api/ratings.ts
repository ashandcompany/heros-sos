import { incidentsMock } from "@/mocks/incidents";
import { ratingsMock } from "@/mocks/ratings";
import type {
  Incident,
  ModerationStatus,
  MutationResult,
  RateableIntervention,
  RatingForModeration,
  ResolvedIncident,
} from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

const isResolved = (incident: Incident): incident is ResolvedIncident =>
  incident.status === "resolu" && !!incident.resolvedAt && !!incident.hero;

/**
 * Interventions terminées sur les incidents déclarés par le citoyen, des plus récentes aux plus
 * anciennes, avec la note déjà donnée le cas échéant.
 * Future implémentation : `GET ${API_URL}/me/interventions`
 */
export async function getMyInterventions(token: string): Promise<RateableIntervention[]> {
  const user = await userFromToken(token, "citoyen");
  const interventions = incidentsMock
    .filter((incident) => incident.reporterId === user.id)
    .filter(isResolved)
    .sort((a, b) => Date.parse(b.resolvedAt) - Date.parse(a.resolvedAt))
    .map((incident) => ({
      incident,
      rating: ratingsMock.find((rating) => rating.incidentId === incident.id) ?? null,
    }));

  return mockResponse(interventions);
}

/**
 * Note le super-héros intervenu sur un incident du citoyen (une seule fois).
 * Future implémentation : `POST ${API_URL}/incidents/${incidentId}/rating` avec `{ rating, comment }`.
 */
export async function rateIntervention(
  token: string,
  incidentId: string,
  { rating, comment }: { rating: number; comment: string | null },
): Promise<MutationResult> {
  const user = await userFromToken(token, "citoyen");
  const incident = incidentsMock.find(({ id }) => id === incidentId);

  if (!incident || incident.reporterId !== user.id || !isResolved(incident)) {
    return mockResponse({ ok: false, error: "Cette intervention ne peut pas être notée." });
  }
  if (ratingsMock.some((existing) => existing.incidentId === incidentId)) {
    return mockResponse({ ok: false, error: "Vous avez déjà noté cette intervention." });
  }

  ratingsMock.push({
    incidentId,
    heroId: incident.hero.id,
    rating,
    comment,
    commentModeration: comment ? "en-attente" : "valide",
    ratedAt: new Date().toISOString(),
  });
  return mockResponse({ ok: true });
}

/* ---------- Back-office (rôle admin) ---------- */

/**
 * Évaluations accompagnées d'un commentaire, des plus récentes aux plus anciennes.
 * Future implémentation : `GET ${API_URL}/admin/ratings?hasComment=true&moderation=${moderation}`
 */
export async function getRatingsForModeration(
  token: string,
  moderation?: ModerationStatus,
): Promise<RatingForModeration[]> {
  await userFromToken(token, "admin");
  const ratings = ratingsMock
    .filter((rating) => rating.comment)
    .filter((rating) => !moderation || rating.commentModeration === moderation)
    .sort((a, b) => Date.parse(b.ratedAt) - Date.parse(a.ratedAt))
    .flatMap((rating) => {
      const incident = incidentsMock.find(({ id }) => id === rating.incidentId);
      if (!incident?.hero) return [];
      return [
        {
          ...rating,
          incident: { id: incident.id, title: incident.title },
          hero: incident.hero,
        },
      ];
    });

  return mockResponse(ratings);
}

/**
 * Publie ou masque le commentaire d'une évaluation (la note reste prise en compte).
 * Future implémentation : `PUT ${API_URL}/admin/ratings/${incidentId}/moderation` avec `{ decision }`.
 */
export async function moderateComment(
  token: string,
  incidentId: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const rating = ratingsMock.find((existing) => existing.incidentId === incidentId);
  if (!rating) return mockResponse({ ok: false, error: "Évaluation introuvable." });

  rating.commentModeration = decision;
  return mockResponse({ ok: true });
}
