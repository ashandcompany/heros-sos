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
}

export type VillainSummary = Pick<Villain, "id" | "alias">;
