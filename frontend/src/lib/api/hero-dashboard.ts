import { incidentsMock } from "@/mocks/incidents";
import { interventionsMock } from "@/mocks/interventions";
import { ratingsMock } from "@/mocks/ratings";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { videosMock } from "@/mocks/videos";
import { villainsMock } from "@/mocks/villains";
import { incidentTypeLabels } from "@/lib/format";
import type { HeroDashboard, IncidentType, VillainSummary } from "@/types";
import { mockResponse } from "./config";
import { summarizeRatings } from "./ratings";
import { userFromToken } from "./session-user";

const MAIN_ENEMIES_LIMIT = 3;
const RECENT_INCIDENTS_LIMIT = 5;

/**
 * Statistiques globales et personnelles du tableau de bord professionnel.
 * Future implémentation : `GET ${API_URL}/hero/dashboard`
 */
export async function getHeroDashboard(token: string): Promise<HeroDashboard> {
  const user = await userFromToken(token, "super-heros");
  const profile = superHeroProfilesMock.find(({ id }) => id === user.id);
  const incidents = incidentsMock.filter(({ moderation }) => moderation !== "rejete");

  const incidentsByType = Object.fromEntries(
    Object.keys(incidentTypeLabels).map((type) => [type, 0]),
  ) as Record<IncidentType, number>;
  for (const { type } of incidents) incidentsByType[type] += 1;

  const myInterventions = interventionsMock.filter(({ hero }) => hero.id === user.id);
  const myVideos = videosMock.filter(({ hero }) => hero.id === user.id);

  // Vilains affrontés : un par incident sur lequel le héros est (ou a été) intervenu.
  const myIncidentIds = new Set([
    ...myInterventions.map(({ incidentId }) => incidentId),
    ...incidents.filter(({ hero }) => hero?.id === user.id).map(({ id }) => id),
  ]);
  const encounters = new Map<string, VillainSummary & { encounters: number }>();
  for (const { id, villain } of incidents) {
    if (!villain || !myIncidentIds.has(id)) continue;
    const entry = encounters.get(villain.id) ?? { ...villain, encounters: 0 };
    entry.encounters += 1;
    encounters.set(villain.id, entry);
  }

  return mockResponse({
    global: {
      interventions: superHeroesMock.reduce((sum, hero) => sum + hero.interventionsCount, 0),
      incidentsByType,
      // Mock : un incident résolu compte pour un citoyen secouru (l'API fournira le chiffre réel).
      rescuedCitizens: incidents.filter(({ status }) => status === "resolu").length,
      villainsIdentified: villainsMock.length,
      villainsImprisoned: villainsMock.filter(({ status }) => status === "emprisonne").length,
    },
    personal: {
      interventions: profile?.interventionsCount ?? myInterventions.length,
      successfulInterventions: myInterventions.filter(({ outcome }) => outcome === "reussi").length,
      failedInterventions: myInterventions.filter(({ outcome }) => outcome === "echoue").length,
      ...summarizeRatings(ratingsMock.filter(({ heroId }) => heroId === user.id)),
      videos: myVideos.length,
      publishedVideos: myVideos.filter(({ moderation }) => moderation === "valide").length,
      mainEnemies: [...encounters.values()]
        .sort((a, b) => b.encounters - a.encounters || a.alias.localeCompare(b.alias, "fr"))
        .slice(0, MAIN_ENEMIES_LIMIT),
    },
    myOngoingIncidents: incidents.filter(
      ({ status, hero }) => status === "en-cours" && hero?.id === user.id,
    ),
    recentIncidents: incidents
      .filter(({ status }) => status !== "resolu")
      .sort((a, b) => Date.parse(b.reportedAt) - Date.parse(a.reportedAt))
      .slice(0, RECENT_INCIDENTS_LIMIT),
  });
}
