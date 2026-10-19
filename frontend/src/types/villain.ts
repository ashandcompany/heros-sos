export type ThreatLevel = "faible" | "modere" | "eleve" | "critique";

export interface Villain {
  id: string;
  alias: string;
  threatLevel: ThreatLevel;
  /** Mode opératoire connu, alimenté par les déclarations et la modération. */
  description: string;
  reportedIncidentsCount: number;
}

export type VillainSummary = Pick<Villain, "id" | "alias">;
