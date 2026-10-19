import type { ModerationStatus } from "./moderation";
import type { SuperHeroSummary } from "./super-hero";

/** Film ajouté par un super-héros dans sa vidéothèque, publié après modération. */
export interface HeroVideo {
  id: string;
  hero: SuperHeroSummary;
  title: string;
  description: string;
  /** Lien vers la vidéo (hébergeur externe). */
  videoUrl: string;
  /** Date ISO 8601 de l'ajout. */
  addedAt: string;
  moderation: ModerationStatus;
}
