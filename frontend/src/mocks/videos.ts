import type { HeroVideo } from "@/types";

/** Films des vidéothèques des super-héros (liens fictifs). */
export const videosMock: HeroVideo[] = [
  {
    id: "video-1",
    hero: { id: "hero-1", alias: "Lumen" },
    title: "Se protéger d'un aveuglement lumineux",
    description: "Les bons réflexes à adopter quand un vilain utilise des flashs aveuglants.",
    videoUrl: "https://videos.exemple.fr/lumen/aveuglement",
    addedAt: "2026-09-28T10:00:00+02:00",
    moderation: "valide",
  },
  {
    id: "video-2",
    hero: { id: "hero-2", alias: "Rempart" },
    title: "Évacuer une rame de métro en sécurité",
    description: "Retour sur l'intervention du 16 octobre et conseils pour garder son calme.",
    videoUrl: "https://videos.exemple.fr/rempart/evacuation-metro",
    addedAt: "2026-10-17T09:30:00+02:00",
    moderation: "en-attente",
  },
  {
    id: "video-3",
    hero: { id: "hero-3", alias: "Vélocité" },
    title: "Mes 10 meilleures courses (avec ma nouvelle boisson énergisante)",
    description: "Code promo VELOCITE20 sur toute la boutique !",
    videoUrl: "https://videos.exemple.fr/velocite/top10-sponsorise",
    addedAt: "2026-10-18T19:12:00+02:00",
    moderation: "en-attente",
  },
];
