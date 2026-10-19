import type { ModerationStatus } from "./moderation";
import type { SuperHeroSummary } from "./super-hero";
import type { VillainSummary } from "./villain";

export type IncidentType =
  "vol" | "cambriolage" | "prise-d-otages" | "agression" | "sabotage" | "enlevement";

export type IncidentStatus = "signale" | "en-cours" | "resolu";

/** Urgence de l'incident, définie par les super-héros. */
export type IncidentPriority = "basse" | "normale" | "haute" | "critique";

/** Résultat d'une intervention terminée. */
export type InterventionOutcome = "reussi" | "echoue";

/** Le citoyen qui déclare est témoin ou victime de l'incident. */
export type ReporterRole = "temoin" | "victime";

export interface IncidentLocation {
  city: string;
  district?: string;
  /** Lieu précis saisi par le déclarant (adresse, repère…). */
  address?: string;
}

export interface Incident {
  id: string;
  title: string;
  type: IncidentType;
  status: IncidentStatus;
  /**
   * Vérification par la modération (déclaration conforme et réelle).
   * Un incident rejeté n'apparaît plus dans les listes publiques.
   */
  moderation: ModerationStatus;
  /** Priorité fixée par un super-héros (`normale` tant qu'elle n'a pas été définie). */
  priority?: IncidentPriority;
  location: IncidentLocation;
  /** Date ISO 8601 du signalement. */
  reportedAt: string;
  /** Date ISO 8601 de résolution, présente uniquement si `status === "resolu"`. */
  resolvedAt?: string;
  /** Vilain impliqué, `null` s'il n'a pas été identifié. */
  villain: VillainSummary | null;
  /** Super-héros intervenu, absent tant que l'incident n'est pas pris en charge. */
  hero?: SuperHeroSummary;
  description?: string;
  /** Date ISO 8601 des faits si elle diffère du signalement. */
  occurredAt?: string;
  /** Citoyen ayant déclaré l'incident. */
  reporterId?: string;
  reporterRole?: ReporterRole;
}

/** Incident traité avec succès : un héros est forcément intervenu. */
export type ResolvedIncident = Incident & {
  status: "resolu";
  resolvedAt: string;
  hero: SuperHeroSummary;
};

/** Données saisies par un citoyen pour déclarer un incident. */
export interface IncidentDeclaration {
  type: IncidentType;
  title: string;
  description: string;
  reporterRole: ReporterRole;
  location: { address: string; city: string };
  /** Date ISO 8601 des faits, `null` si l'incident est en cours. */
  occurredAt: string | null;
  /** `null` si le citoyen ne sait pas quel vilain est impliqué. */
  villainId: string | null;
}

/** Évaluation d'une intervention par le citoyen impliqué. */
export interface InterventionRating {
  incidentId: string;
  heroId: string;
  /** Note entière de 1 à 5. */
  rating: number;
  comment: string | null;
  /** Modération du commentaire : un commentaire rejeté est masqué (la note reste comptée). */
  commentModeration: ModerationStatus;
  ratedAt: string;
  /** Réponse du super-héros au commentaire. */
  reply?: HeroReply | null;
}

export interface HeroReply {
  content: string;
  /** Date ISO 8601. */
  repliedAt: string;
}

/** Évaluation affichée dans le back-office, avec l'incident et le héros concernés. */
export interface RatingForModeration extends InterventionRating {
  incident: Pick<Incident, "id" | "title">;
  hero: SuperHeroSummary;
}

/** Modifications d'un incident par la modération. */
export interface IncidentUpdate {
  title: string;
  type: IncidentType;
  status: IncidentStatus;
  description: string;
  location: { address: string; city: string };
  villainId: string | null;
  /** Obligatoire dès que l'incident est pris en charge. */
  heroId: string | null;
}

/** Intervention terminée sur un incident du citoyen, avec sa note s'il l'a déjà donnée. */
export interface RateableIntervention {
  incident: ResolvedIncident;
  rating: InterventionRating | null;
}

/**
 * Intervention d'un super-héros sur un incident. Un seul héros intervient à la fois :
 * après un échec, l'incident redevient disponible pour une nouvelle prise en charge.
 */
export interface HeroIntervention {
  id: string;
  incidentId: string;
  hero: SuperHeroSummary;
  /** Dates ISO 8601. */
  startedAt: string;
  endedAt: string | null;
  /** `null` tant que l'intervention est en cours. */
  outcome: InterventionOutcome | null;
}

/** Incident vu depuis l'espace super-héros : priorité et historique des interventions. */
export interface HeroIncident extends Incident {
  priority: IncidentPriority;
  /** Interventions sur cet incident, des plus récentes aux plus anciennes. */
  interventions: HeroIntervention[];
}

/**
 * Avis affiché sur la fiche publique d'un super-héros : uniquement les commentaires publiés
 * par la modération, sans aucune information sur le citoyen qui l'a écrit.
 */
export interface PublicReview {
  incidentId: string;
  rating: number;
  comment: string;
  ratedAt: string;
  reply: HeroReply | null;
}
