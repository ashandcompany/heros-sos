import type { InterventionRating } from "@/types";

export const ratingsMock: InterventionRating[] = [
  {
    incidentId: "incident-6",
    heroId: "hero-8",
    rating: 5,
    comment: "Intervention très rapide, Titanide a tout sécurisé en quelques minutes.",
    commentModeration: "valide",
    ratedAt: "2026-10-12T19:02:00+02:00",
  },
  {
    incidentId: "incident-2",
    heroId: "hero-2",
    rating: 4,
    comment: "Rempart a protégé tout le wagon, mais l'attente a été longue avant son arrivée.",
    commentModeration: "en-attente",
    ratedAt: "2026-10-16T12:30:00+02:00",
  },
  {
    incidentId: "incident-3",
    heroId: "hero-6",
    rating: 1,
    comment: "Nulle, incompétente, elle devrait rendre sa cape. Appelez-moi au 06 99 88 77 66.",
    commentModeration: "en-attente",
    ratedAt: "2026-10-16T08:10:00+02:00",
  },
  {
    incidentId: "incident-1",
    heroId: "hero-1",
    rating: 5,
    comment: "Lumen a aveuglé le Corbeau Gris en une seconde, personne n'a été blessé. Merci !",
    commentModeration: "valide",
    ratedAt: "2026-10-17T16:05:00+02:00",
    reply: null,
  },
  {
    incidentId: "incident-13",
    heroId: "hero-1",
    rating: 4,
    comment:
      "Elle a éclairé tout le tunnel pour nous guider vers la sortie. Un peu d'attente au début.",
    commentModeration: "valide",
    ratedAt: "2026-10-10T08:40:00+02:00",
    reply: {
      content:
        "Merci pour votre patience ! J'étais encore sur une autre intervention à la Croix-Rousse.",
      repliedAt: "2026-10-10T12:15:00+02:00",
    },
  },
  {
    incidentId: "incident-14",
    heroId: "hero-1",
    rating: 3,
    comment:
      "Les statues ont été retrouvées, mais deux vitrines ont été brisées pendant l'intervention.",
    commentModeration: "en-attente",
    ratedAt: "2026-10-03T09:00:00+02:00",
  },
];
