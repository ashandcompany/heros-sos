import type { ApiFailure } from "./api";

export type ThreatLevel = "faible" | "modere" | "eleve" | "critique";

export type VillainStatus = "en-liberte" | "emprisonne";

export interface Villain {
  id: string;
  alias: string;
  threatLevel: ThreatLevel;
  /** Mode opératoire connu, alimenté par les déclarations et la modération. */
  description: string;
  reportedIncidentsCount: number;
  status: VillainStatus;
  /** État civil et secteur, facultatifs (chaînes vides si inconnus). */
  lastName: string;
  firstName: string;
  mainSector: string;
  /** Spécialités : identifiants du référentiel des types d'incident. */
  typeIds: number[];
  /** Les mêmes spécialités, détaillées pour l'affichage, par libellé. */
  specialties: VillainSpecialty[];
}

export interface VillainSpecialty {
  id: number;
  code: string;
  label: string;
}

export type VillainSummary = Pick<Villain, "id" | "alias">;

/** Saisie d'une fiche de vilain par la modération (`VillainPayload` / `VillainUpdatePayload`). */
export interface VillainInput {
  alias: string;
  lastName: string;
  firstName: string;
  mainSector: string;
  description: string;
  typeIds: number[];
  threatLevel: ThreatLevel;
  status: VillainStatus;
}

export type VillainSaveResult =
  | { ok: true; villain: Villain }
  | { ok: false; error: "alias-already-used" }
  | { ok: false; error: "not-found" }
  | { ok: false; error: "invalid-fields"; fields: Record<string, string> }
  | ApiFailure;
