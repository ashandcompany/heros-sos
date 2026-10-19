import type { SuperHeroSummary } from "./super-hero";
import type { VillainSummary } from "./villain";

export type IncidentType =
  "vol" | "cambriolage" | "prise-d-otages" | "agression" | "sabotage" | "enlevement";

export type IncidentStatus = "signale" | "en-cours" | "resolu";

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
  ratedAt: string;
}

/** Intervention terminée sur un incident du citoyen, avec sa note s'il l'a déjà donnée. */
export interface RateableIntervention {
  incident: ResolvedIncident;
  rating: InterventionRating | null;
}
