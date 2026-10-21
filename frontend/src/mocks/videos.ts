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
  {
    id: "video-4",
    hero: { id: "hero-1", alias: "Lumen" },
    title: "Lumen : La Nuit des Corbeaux",
    description:
      "Lyon plonge dans le noir et une héroïne de lumière traque une nuée d'oiseaux mécaniques.",
    videoUrl: "https://www.imdb.com/title/tt9900101/",
    imdbId: "tt9900101",
    year: 2025,
    addedAt: "2026-10-15T18:00:00+02:00",
    moderation: "valide",
  },
  {
    id: "video-5",
    hero: { id: "hero-1", alias: "Lumen" },
    title: "Lumen 2 : Éclipse",
    description:
      "Face à un vilain capable d'absorber la lumière, Lumen doit apprendre à se battre dans l'ombre.",
    videoUrl: "https://www.imdb.com/title/tt9900102/",
    imdbId: "tt9900102",
    year: 2026,
    addedAt: "2026-10-18T22:30:00+02:00",
    moderation: "en-attente",
  },
];
