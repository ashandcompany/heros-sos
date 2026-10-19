import { incidentsMock } from "@/mocks/incidents";
import { superHeroesMock } from "@/mocks/super-heroes";
import { villainsMock } from "@/mocks/villains";
import type {
  Incident,
  IncidentDeclaration,
  IncidentUpdate,
  ModerationStatus,
  MutationResult,
  ResolvedIncident,
} from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

function isResolved(incident: Incident): incident is ResolvedIncident {
  return incident.status === "resolu" && !!incident.resolvedAt && !!incident.hero;
}

/** Les incidents rejetés par la modération (faux signalements) ne sont jamais diffusés. */
const isPublic = (incident: Incident) => incident.moderation !== "rejete";

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
    .filter(isPublic)
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
    .filter(isPublic)
    .filter((incident) => incident.status !== "resolu")
    .filter((incident) => !wanted || normalizeCity(incident.location.city) === wanted)
    .sort(byMostRecent);

  return mockResponse(ongoing);
}

/**
 * Incidents déclarés par le citoyen connecté, du plus récent au plus ancien
 * (y compris ceux rejetés par la modération, pour qu'il en soit informé).
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
 * Un citoyen ou un modérateur peut déclarer : la déclaration d'un modérateur est validée d'office,
 * celle d'un citoyen attend la vérification de la modération.
 * Future implémentation : `POST ${API_URL}/incidents` avec l'`IncidentDeclaration`.
 */
export async function declareIncident(
  token: string,
  declaration: IncidentDeclaration,
): Promise<MutationResult> {
  const user = await userFromToken(token, ["citoyen", "admin"]);
  const villain = villainsMock.find(({ id }) => id === declaration.villainId);

  incidentsMock.push({
    id: `incident-${crypto.randomUUID()}`,
    title: declaration.title,
    type: declaration.type,
    status: "signale",
    moderation: user.role === "admin" ? "valide" : "en-attente",
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

/* ---------- Back-office (rôle admin) ---------- */

/**
 * Tous les incidents, éventuellement filtrés par décision de modération,
 * du plus récent au plus ancien.
 * Future implémentation : `GET ${API_URL}/admin/incidents?moderation=${moderation}`
 */
export async function getAllIncidents(
  token: string,
  moderation?: ModerationStatus,
): Promise<Incident[]> {
  await userFromToken(token, "admin");
  return mockResponse(
    incidentsMock
      .filter((incident) => !moderation || incident.moderation === moderation)
      .sort(byMostRecent),
  );
}

/** Future implémentation : `GET ${API_URL}/admin/incidents/${id}` (404 → `null`). */
export async function getIncident(token: string, id: string): Promise<Incident | null> {
  await userFromToken(token, "admin");
  return mockResponse(incidentsMock.find((incident) => incident.id === id) ?? null);
}

/**
 * Corrige les informations d'un incident (titre, lieu, vilain, héros, statut…).
 * Future implémentation : `PATCH ${API_URL}/admin/incidents/${id}` avec l'`IncidentUpdate`.
 */
export async function updateIncident(
  token: string,
  id: string,
  update: IncidentUpdate,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const incident = incidentsMock.find((existing) => existing.id === id);
  if (!incident) return mockResponse({ ok: false, error: "Incident introuvable." });

  const villain = villainsMock.find(({ id: villainId }) => villainId === update.villainId);
  const hero = superHeroesMock.find(({ id: heroId }) => heroId === update.heroId);
  if (update.status !== "signale" && !hero) {
    return mockResponse({
      ok: false,
      error: "Un incident pris en charge ou résolu doit avoir un super-héros.",
    });
  }

  incident.title = update.title;
  incident.type = update.type;
  incident.description = update.description;
  incident.location = { ...incident.location, ...update.location };
  incident.villain = villain ? { id: villain.id, alias: villain.alias } : null;
  incident.hero =
    hero && update.status !== "signale" ? { id: hero.id, alias: hero.alias } : undefined;
  if (update.status === "resolu" && incident.status !== "resolu") {
    incident.resolvedAt = new Date().toISOString();
  }
  if (update.status !== "resolu") incident.resolvedAt = undefined;
  incident.status = update.status;

  return mockResponse({ ok: true });
}

/**
 * Valide (déclaration conforme et réelle) ou rejette un incident.
 * Future implémentation : `PUT ${API_URL}/admin/incidents/${id}/moderation` avec `{ decision }`.
 */
export async function moderateIncident(
  token: string,
  id: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const incident = incidentsMock.find((existing) => existing.id === id);
  if (!incident) return mockResponse({ ok: false, error: "Incident introuvable." });

  incident.moderation = decision;
  return mockResponse({ ok: true });
}
