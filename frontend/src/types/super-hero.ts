import type { Civility, PostalAddress } from "./auth";
import type { ModerationStatus } from "./moderation";

/** Demande d'inscription d'un nouveau super-héros, validée ou refusée par la modération. */
export interface HeroApplication {
  id: string;
  alias: string;
  power: string;
  civility: Civility;
  firstName: string;
  lastName: string;
  /** Date ISO `YYYY-MM-DD`. */
  birthDate: string;
  email: string;
  mobilePhone: string;
  address: PostalAddress;
  /** Présentation et motivations du candidat. */
  motivation: string;
  /** Date ISO 8601 du dépôt de la demande. */
  submittedAt: string;
  status: ModerationStatus;
  /** Date ISO 8601 de la décision, `null` tant que la demande est en attente. */
  reviewedAt: string | null;
}

export interface SuperHero {
  id: string;
  /** Nom de super-héros affiché publiquement. */
  alias: string;
  /** URL de la photo de profil. `null` tant qu'aucune photo n'est fournie (placeholder affiché). */
  avatarUrl: string | null;
  /** Pouvoir principal, en une courte phrase. */
  power: string;
  /** Note moyenne attribuée par les citoyens, de 0 à 5. */
  averageRating: number;
  interventionsCount: number;
}

/** Version allégée d'un super-héros, embarquée dans d'autres ressources (ex : un incident). */
export type SuperHeroSummary = Pick<SuperHero, "id" | "alias">;

/**
 * Fiche complète d'un super-héros, avec ses données personnelles.
 * Réservée au back-office : ne jamais l'exposer sur une page publique.
 */
export interface SuperHeroProfile extends SuperHero {
  /** Identité civile : `null` si le héros ne l'a pas communiquée. */
  firstName: string | null;
  lastName: string | null;
  civility: Civility | null;
  /** Date ISO `YYYY-MM-DD`, `null` si inconnue. */
  birthDate: string | null;
  email: string;
  mobilePhone: string;
  address: PostalAddress | null;
  /** Date ISO 8601 de l'inscription sur la plateforme. */
  registeredAt: string;
  /** Date ISO 8601 de la dernière intervention, `null` si aucune. */
  lastInterventionAt: string | null;
}
