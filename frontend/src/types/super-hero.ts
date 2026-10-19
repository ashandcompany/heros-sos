import type { Civility, PostalAddress } from "./auth";
import type { Incident, IncidentType, InterventionRating } from "./incident";
import type { ModerationStatus } from "./moderation";
import type { VillainSummary } from "./villain";

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
  /** Nouvelle adresse en attente de validation par la modération. */
  pendingEmail: string | null;
  mobilePhone: string;
  address: PostalAddress | null;
  /** Date ISO 8601 de l'inscription sur la plateforme. */
  registeredAt: string;
  /** Date ISO 8601 de la dernière intervention, `null` si aucune. */
  lastInterventionAt: string | null;
}

/** Tableau de bord professionnel d'un super-héros. */
export interface HeroDashboard {
  /** Statistiques de toute la plateforme. */
  global: {
    interventions: number;
    incidentsByType: Record<IncidentType, number>;
    rescuedCitizens: number;
    villainsIdentified: number;
    villainsImprisoned: number;
  };
  personal: {
    interventions: number;
    successfulInterventions: number;
    failedInterventions: number;
    averageRating: number;
    ratingsCount: number;
    videos: number;
    publishedVideos: number;
    /** Vilains les plus souvent affrontés, du plus fréquent au moins fréquent. */
    mainEnemies: Array<VillainSummary & { encounters: number }>;
  };
  /** Incidents sur lesquels le héros intervient en ce moment. */
  myOngoingIncidents: Incident[];
  /** Derniers incidents signalés et non résolus, tous héros confondus. */
  recentIncidents: Incident[];
}

/**
 * Évaluation reçue par le héros. `comment` vaut `null` tant que la modération ne l'a pas publié
 * (ou s'il a été masqué) : `commentModeration` permet d'afficher la raison.
 */
export interface ReceivedRating extends InterventionRating {
  incident: Pick<Incident, "id" | "title">;
}

export interface HeroReputation {
  /** Note globale sur 5. */
  averageRating: number;
  ratingsCount: number;
  ratings: ReceivedRating[];
}
