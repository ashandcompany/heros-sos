import { incidentsMock } from "@/mocks/incidents";
import { villainsMock } from "@/mocks/villains";
import type { Incident, IncidentDeclaration, MutationResult, ResolvedIncident } from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

function isResolved(incident: Incident): incident is ResolvedIncident {
  return incident.status === "resolu" && !!incident.resolvedAt && !!incident.hero;
}

const byMostRecent = (a: Incident, b: Incident) =>
  Date.parse(b.reportedAt) - Date.parse(a.reportedAt);

/** Compare deux noms de ville sans tenir compte de la casse ni des accents. */
const normalizeCity = (city: string) =>
  city
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .trim()
    .toLowerCase();

/**
 * Derniers incidents traités avec succès, du plus récent au plus ancien.
 * Future implémentation : `GET ${API_URL}/incidents?status=resolu&sort=-resolvedAt&limit=${limit}`
 */
export async function getLatestResolvedIncidents(limit = 6): Promise<ResolvedIncident[]> {
  const resolved = incidentsMock
    .filter(isResolved)
    .sort((a, b) => Date.parse(b.resolvedAt) - Date.parse(a.resolvedAt))
    .slice(0, limit);

  return mockResponse(resolved);
}

/**
 * Incidents non résolus (signalés ou pris en charge), éventuellement filtrés par ville.
 * Future implémentation : `GET ${API_URL}/incidents?status=signale,en-cours&city=${city}`
 */
export async function getOngoingIncidents(city?: string): Promise<Incident[]> {
  const wanted = city ? normalizeCity(city) : null;
  const ongoing = incidentsMock
    .filter((incident) => incident.status !== "resolu")
    .filter((incident) => !wanted || normalizeCity(incident.location.city) === wanted)
    .sort(byMostRecent);

  return mockResponse(ongoing);
}

/**
 * Incidents déclarés par le citoyen connecté, du plus récent au plus ancien.
 * Future implémentation : `GET ${API_URL}/me/incidents`
 */
export async function getMyIncidents(token: string): Promise<Incident[]> {
  const user = await userFromToken(token, "citoyen");
  return mockResponse(
    incidentsMock.filter((incident) => incident.reporterId === user.id).sort(byMostRecent),
  );
}

/**
 * Déclare un incident ; il est diffusé aux super-héros à proximité.
 * Future implémentation : `POST ${API_URL}/incidents` avec l'`IncidentDeclaration`.
 */
export async function declareIncident(
  token: string,
  declaration: IncidentDeclaration,
): Promise<MutationResult> {
  const user = await userFromToken(token, "citoyen");
  const villain = villainsMock.find(({ id }) => id === declaration.villainId);

  incidentsMock.push({
    id: `incident-${crypto.randomUUID()}`,
    title: declaration.title,
    type: declaration.type,
    status: "signale",
    location: declaration.location,
    reportedAt: new Date().toISOString(),
    occurredAt: declaration.occurredAt ?? undefined,
    villain: villain ? { id: villain.id, alias: villain.alias } : null,
    description: declaration.description,
    reporterId: user.id,
    reporterRole: declaration.reporterRole,
  });

  return mockResponse({ ok: true });
}
