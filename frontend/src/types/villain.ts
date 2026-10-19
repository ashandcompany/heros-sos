export type ThreatLevel = "faible" | "modere" | "eleve" | "critique";

export interface Villain {
  id: string;
  alias: string;
  threatLevel: ThreatLevel;
}

export type VillainSummary = Pick<Villain, "id" | "alias">;
