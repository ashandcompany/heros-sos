import type { SuperHeroSummary } from "./super-hero";
import type { VillainSummary } from "./villain";

export type IncidentType =
  "vol" | "cambriolage" | "prise-d-otages" | "agression" | "sabotage" | "enlevement";

export type IncidentStatus = "signale" | "en-cours" | "resolu";

export interface IncidentLocation {
  city: string;
  district?: string;
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
  villain: VillainSummary;
  /** Super-héros intervenu, absent tant que l'incident n'est pas pris en charge. */
  hero?: SuperHeroSummary;
}

/** Incident traité avec succès : un héros est forcément intervenu. */
export type ResolvedIncident = Incident & {
  status: "resolu";
  resolvedAt: string;
  hero: SuperHeroSummary;
};
