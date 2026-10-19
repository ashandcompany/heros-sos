import type { ModerationStatus } from "./moderation";
import type { SuperHeroSummary } from "./super-hero";

/** Film ajouté par un super-héros dans sa vidéothèque, publié après modération. */
export interface HeroVideo {
  id: string;
  hero: SuperHeroSummary;
  title: string;
  description: string;
  /** Lien vers la vidéo (hébergeur externe ou fiche IMDb). */
  videoUrl: string;
  /** Identifiant IMDb (`tt…`) quand le film provient du catalogue IMDb. */
  imdbId?: string;
  year?: number;
  /** Date ISO 8601 de l'ajout. */
  addedAt: string;
  moderation: ModerationStatus;
}

/** Résultat d'une recherche dans le catalogue IMDb. */
export interface ImdbTitle {
  /** Identifiant IMDb, ex : `tt0000001`. */
  id: string;
  title: string;
  year: number;
  plot: string;
}
