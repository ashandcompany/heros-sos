import { incidentsMock } from "@/mocks/incidents";
import { interventionsMock } from "@/mocks/interventions";
import { reviewsMock } from "@/mocks/reviews";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { videosMock } from "@/mocks/videos";
import type { HeroDashboard, VillainSummary } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { type ApiSchema, apiFetch } from "./http";
import { getIncidentTypes } from "./incident-types";
import { withTeams } from "./incidents";
import { sessionUserId } from "./interventions";
import { summarizeRatings } from "./reviews";
import { userFromToken } from "./session-user";
import { getVillains } from "./villains";

const MAIN_ENEMIES_LIMIT = 3;
const RECENT_INCIDENTS_LIMIT = 5;

/**
 * Tableau de bord professionnel : statistiques globales et personnelles, incidents en cours du
 * héros et derniers incidents non résolus.
 * API : `GET /hero/dashboard`, puis l'équipe de chaque incident en cours (`GET /interventions/{id}`).
 */
export async function getHeroDashboard(token: string): Promise<HeroDashboard> {
  if (isApiEnabled) {
    const [response, meId] = await Promise.all([
      apiFetch("/hero/dashboard", { sessionToken: token }),
      sessionUserId(token),
    ]);
    const dashboard = (await response.json()) as ApiSchema<"HeroDashboard">;
    const [myOngoingIncidents, recentIncidents] = await Promise.all([
      withTeams(token, dashboard.myOngoingIncidents, meId),
      withTeams(token, dashboard.recentIncidents, meId),
    ]);
    // Statistiques : même forme que `HeroDashboard`, l'API n'a rien à convertir.
    return {
      global: dashboard.global,
      personal: dashboard.personal,
      myOngoingIncidents,
      recentIncidents,
    };
  }

  const user = await userFromToken(token, "super-heros");
  const profile = superHeroProfilesMock.find(({ id }) => id === user.id);
  const incidents = incidentsMock.filter(({ moderation }) => moderation !== "rejete");

  // Tous les types du référentiel, y compris ceux qui n'ont encore aucun signalement.
  const counts = new Map<string, number>();
  for (const { type } of incidents) counts.set(type.code, (counts.get(type.code) ?? 0) + 1);
  const incidentsByType = (await getIncidentTypes(token))
    .map(({ code, label }) => ({ type: { code, label }, count: counts.get(code) ?? 0 }))
    .sort((a, b) => b.count - a.count);

  const villains = await getVillains();
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
      villainsIdentified: villains.length,
      villainsImprisoned: villains.filter(({ status }) => status === "emprisonne").length,
    },
    personal: {
      interventions: profile?.interventionsCount ?? myInterventions.length,
      successfulInterventions: myInterventions.filter(({ outcome }) => outcome === "reussi").length,
      failedInterventions: myInterventions.filter(({ outcome }) => outcome === "echoue").length,
      ...summarizeRatings(reviewsMock.filter(({ hero }) => hero.id === user.id)),
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
