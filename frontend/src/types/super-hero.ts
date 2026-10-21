import type { ApiFailure } from "./api";
import type { Civility, PostalAddress } from "./auth";
import type { Incident, IncidentType } from "./incident";
import type { ModerationStatus } from "./moderation";
import type { VillainSummary } from "./villain";

/**
 * Demande de statut super-héros déposée par un citoyen, vue par la modération (schéma
 * `HeroApplicant`). Un compte n'a qu'une demande : elle est identifiée par son demandeur.
 */
export interface HeroApplication {
  /** Identifiant du compte demandeur (routes de décision). */
  userId: string;
  alias: string;
  /** Description publique proposée (peut être vide). */
  bio: string;
  /** Message adressé aux modérateurs, jamais public (peut être vide). */
  requestMessage: string;
  firstName: string;
  lastName: string;
  email: string;
  /** Date ISO 8601 du dépôt de la demande. */
  submittedAt: string;
  status: ModerationStatus;
}

/** Ce que le citoyen saisit pour demander le statut de super-héros (`HeroRequestPayload`). */
export interface HeroRequestInput {
  alias: string;
  bio: string;
  requestMessage: string;
}

/** État de la demande du citoyen connecté (`HeroRequestStatus`). */
export interface MyHeroRequest {
  status: ModerationStatus;
  alias: string;
  /** Motif communiqué par la modération, seulement pour une demande refusée. */
  rejectionReason: string | null;
}

export type HeroRequestResult =
  | { ok: true }
  | {
      ok: false;
      error:
        | "already-pending"
        | "already-hero"
        | "alias-already-used"
        /** Compte qui ne peut pas faire de demande (désactivé, non citoyen…). */
        | "not-eligible";
    }
  /** Champs refusés par l'API : champ (`alias`, `bio`, `requestMessage`) → code (`invalid`…). */
  | { ok: false; error: "invalid-fields"; fields: Record<string, string> }
  | ApiFailure;

export type HeroReviewResult =
  | { ok: true }
  | { ok: false; error: "not-found" | "already-reviewed" }
  | { ok: false; error: "invalid-reason" }
  | ApiFailure;

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
    /** Incidents par type, du plus fréquent au moins fréquent. */
    incidentsByType: Array<{ type: IncidentType; count: number }>;
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
