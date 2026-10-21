import type { HeroReview } from "@/types";

/** Avis mock, rattachés à l'incident noté (l'API ne renvoie que l'intervention). */
export type MockReview = HeroReview & { incidentId: string };

/**
 * Intervention mock d'un incident résolu : celle de l'historique si elle existe, sinon un
 * identifiant dérivé de l'incident (les incidents mock ne sont pas tous liés à une intervention).
 */
export function mockInterventionId(incidentId: string): string {
  return (
    {
      "incident-14": "intervention-1",
      "incident-13": "intervention-2",
      "incident-1": "intervention-3",
    }[incidentId] ?? `intervention-${incidentId}`
  );
}

export const reviewsMock: MockReview[] = [
  {
    id: "review-1",
    incidentId: "incident-6",
    interventionId: mockInterventionId("incident-6"),
    hero: { id: "hero-8", alias: "Titanide" },
    rating: 5,
    comment: "Intervention très rapide, Titanide a tout sécurisé en quelques minutes.",
    status: "valide",
    createdAt: "2026-10-12T19:02:00+02:00",
    reply: null,
  },
  {
    id: "review-2",
    incidentId: "incident-2",
    interventionId: mockInterventionId("incident-2"),
    hero: { id: "hero-2", alias: "Rempart" },
    rating: 4,
    comment: "Rempart a protégé tout le wagon, mais l'attente a été longue avant son arrivée.",
    status: "valide",
    createdAt: "2026-10-16T12:30:00+02:00",
    reply: {
      content:
        "Désolé pour l'attente : le trafic était coupé sur tout le pont. Merci de votre avis !",
      status: "en-attente",
      createdAt: "2026-10-16T18:00:00+02:00",
      editedAt: "2026-10-17T09:12:00+02:00",
    },
  },
  {
    id: "review-3",
    incidentId: "incident-3",
    interventionId: mockInterventionId("incident-3"),
    hero: { id: "hero-6", alias: "Magnéta" },
    rating: 1,
    comment: "Nulle, incompétente, elle devrait rendre sa cape. Appelez-moi au 06 99 88 77 66.",
    status: "rejete",
    createdAt: "2026-10-16T08:10:00+02:00",
    reply: null,
  },
  {
    id: "review-4",
    incidentId: "incident-1",
    interventionId: mockInterventionId("incident-1"),
    hero: { id: "hero-1", alias: "Lumen" },
    rating: 5,
    comment: "Lumen a aveuglé le Corbeau Gris en une seconde, personne n'a été blessé. Merci !",
    status: "valide",
    createdAt: "2026-10-17T16:05:00+02:00",
    reply: null,
  },
  {
    id: "review-5",
    incidentId: "incident-13",
    interventionId: mockInterventionId("incident-13"),
    hero: { id: "hero-1", alias: "Lumen" },
    rating: 4,
    comment:
      "Elle a éclairé tout le tunnel pour nous guider vers la sortie. Un peu d'attente au début.",
    status: "valide",
    createdAt: "2026-10-10T08:40:00+02:00",
    reply: {
      content:
        "Merci pour votre patience ! J'étais encore sur une autre intervention à la Croix-Rousse.",
      status: "valide",
      createdAt: "2026-10-10T12:15:00+02:00",
      editedAt: null,
    },
  },
  {
    id: "review-6",
    incidentId: "incident-14",
    interventionId: mockInterventionId("incident-14"),
    hero: { id: "hero-1", alias: "Lumen" },
    rating: 3,
    comment:
      "Les statues ont été retrouvées, mais deux vitrines ont été brisées pendant l'intervention.",
    status: "valide",
    createdAt: "2026-10-03T09:00:00+02:00",
    reply: null,
  },
];
