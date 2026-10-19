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
];
