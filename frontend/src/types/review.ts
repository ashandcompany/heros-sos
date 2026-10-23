import type { InterventionOutcome } from "./incident";
import type { ModerationStatus } from "./moderation";
import type { SuperHeroSummary } from "./super-hero";

/*
 * Avis sur les super-héros. Le déclarant d'un incident note chaque héros confirmé d'une
 * intervention terminée (réussie ou non), une seule fois par héros.
 */

/**
 * Statut d'un avis : publié dès son envoi (`valide`). Masqué par la modération (`rejete`), il
 * disparaît des pages publiques et ne compte plus dans la note moyenne du héros.
 */
export type ReviewStatus = ModerationStatus;

/**
 * Réponse du super-héros à un avis : publiée à sa création (`valide`), remise `en-attente` de
 * vérification à chaque modification, `rejete` si la modération la masque.
 */
export interface ReviewReply {
  content: string;
  status: ModerationStatus;
  /** Dates ISO 8601. */
  createdAt: string;
  editedAt: string | null;
}

export interface HeroReview {
  id: string;
  interventionId: string;
  hero: SuperHeroSummary;
  /** Note entière de 1 à 5. */
  rating: number;
  /** `null` sans commentaire, ou quand l'avis est masqué (vu par le héros). */
  comment: string | null;
  status: ReviewStatus;
  /** Date ISO 8601 de l'envoi. */
  createdAt: string;
  reply: ReviewReply | null;
}

/**
 * Super-héros à noter pour une intervention terminée sur un incident déclaré par l'utilisateur :
 * une entrée par héros confirmé, avec l'avis déjà donné le cas échéant.
 */
export interface RateableIntervention {
  interventionId: string;
  /** `label` : type et ville de l'incident, `null` s'il n'a pas pu être lu. */
  incident: { id: string; label: string | null };
  outcome: InterventionOutcome;
  /** Date ISO 8601 de la fin de l'intervention. */
  endedAt: string;
  hero: SuperHeroSummary;
  review: HeroReview | null;
}

/** Note globale et avis reçus par le héros connecté, des plus récents aux plus anciens. */
export interface HeroReputation {
  /** Moyenne sur 5 des avis publiés. */
  averageRating: number;
  /** Nombre d'avis publiés (les avis masqués ne comptent pas). */
  ratingsCount: number;
  /** Tous les avis reçus, masqués compris (sans leur commentaire). */
  reviews: HeroReview[];
}

/** Réponse d'un super-héros à modérer, avec l'avis auquel elle répond. */
export interface ReplyForModeration {
  reviewId: string;
  heroAlias: string;
  rating: number;
  comment: string | null;
  reviewStatus: ReviewStatus;
  reply: ReviewReply;
}
