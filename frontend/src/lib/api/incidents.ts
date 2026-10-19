import { incidentsMock } from "@/mocks/incidents";
import { interventionsMock } from "@/mocks/interventions";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { villainsMock } from "@/mocks/villains";
import type {
  HeroIncident,
  Incident,
  IncidentDeclaration,
  IncidentPriority,
  IncidentStatus,
  IncidentUpdate,
  InterventionOutcome,
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
  const user = await userFromToken(token, ["citoyen", "super-heros"]);
  return mockResponse(
    incidentsMock.filter((incident) => incident.reporterId === user.id).sort(byMostRecent),
  );
}

/**
 * Déclare un incident ; il est diffusé aux super-héros à proximité.
 * Un citoyen, un super-héros ou un modérateur peut déclarer : la déclaration d'un modérateur est validée d'office,
 * celle d'un citoyen ou d'un super-héros attend la vérification de la modération.
 * Future implémentation : `POST ${API_URL}/incidents` avec l'`IncidentDeclaration`.
 */
export async function declareIncident(
  token: string,
  declaration: IncidentDeclaration,
): Promise<MutationResult> {
  const user = await userFromToken(token, ["citoyen", "super-heros", "admin"]);
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

/* ---------- Espace super-héros (rôle super-heros) ---------- */

const priorityRank: Record<IncidentPriority, number> = {
  critique: 0,
  haute: 1,
  normale: 2,
  basse: 3,
};

const statusRank: Record<IncidentStatus, number> = { signale: 0, "en-cours": 1, resolu: 2 };

function toHeroIncident(incident: Incident): HeroIncident {
  return {
    ...incident,
    priority: incident.priority ?? "normale",
    interventions: interventionsMock
      .filter(({ incidentId }) => incidentId === incident.id)
      .sort((a, b) => Date.parse(b.startedAt) - Date.parse(a.startedAt)),
  };
}

/**
 * Signalements diffusés aux super-héros, triés par statut (à prendre en charge d'abord),
 * puis par priorité et du plus récent au plus ancien.
 * Future implémentation : `GET ${API_URL}/hero/incidents?status=${statuses}`
 */
export async function getHeroIncidents(
  token: string,
  statuses?: readonly IncidentStatus[],
): Promise<HeroIncident[]> {
  await userFromToken(token, "super-heros");
  const incidents = incidentsMock
    .filter(isPublic)
    .filter((incident) => !statuses || statuses.includes(incident.status))
    .map(toHeroIncident)
    .sort(
      (a, b) =>
        statusRank[a.status] - statusRank[b.status] ||
        priorityRank[a.priority] - priorityRank[b.priority] ||
        byMostRecent(a, b),
    );

  return mockResponse(incidents);
}

/** Future implémentation : `GET ${API_URL}/hero/incidents/${id}` (404 → `null`). */
export async function getHeroIncident(token: string, id: string): Promise<HeroIncident | null> {
  await userFromToken(token, "super-heros");
  const incident = incidentsMock.find((existing) => existing.id === id && isPublic(existing));
  return mockResponse(incident ? toHeroIncident(incident) : null);
}

/** Incident diffusé et non résolu, sur lequel un héros peut agir. */
function findActionableIncident(id: string): Incident | { error: string } {
  const incident = incidentsMock.find((existing) => existing.id === id && isPublic(existing));
  if (!incident) return { error: "Incident introuvable." };
  if (incident.status === "resolu") return { error: "Cet incident est déjà résolu." };
  return incident;
}

/** Future implémentation : `PUT ${API_URL}/hero/incidents/${id}/priority` avec `{ priority }`. */
export async function setIncidentPriority(
  token: string,
  id: string,
  priority: IncidentPriority,
): Promise<MutationResult> {
  await userFromToken(token, "super-heros");
  const incident = findActionableIncident(id);
  if ("error" in incident) return mockResponse({ ok: false, error: incident.error });

  incident.priority = priority;
  return mockResponse({ ok: true });
}

/**
 * Prise en charge d'un incident : un seul super-héros peut intervenir à la fois
 * (l'API répondra 409 si un autre héros l'a pris en charge entre-temps).
 * Future implémentation : `POST ${API_URL}/hero/incidents/${id}/intervention`
 */
export async function takeChargeOfIncident(token: string, id: string): Promise<MutationResult> {
  const user = await userFromToken(token, "super-heros");
  const incident = findActionableIncident(id);
  if ("error" in incident) return mockResponse({ ok: false, error: incident.error });
  if (incident.status === "en-cours" || incident.hero) {
    return mockResponse({
      ok: false,
      error:
        incident.hero?.id === user.id
          ? "Vous intervenez déjà sur cet incident."
          : "Un autre super-héros intervient déjà sur cet incident.",
    });
  }

  const hero = { id: user.id, alias: user.displayName };
  incident.status = "en-cours";
  incident.hero = hero;
  interventionsMock.push({
    id: `intervention-${crypto.randomUUID()}`,
    incidentId: incident.id,
    hero,
    startedAt: new Date().toISOString(),
    endedAt: null,
    outcome: null,
  });
  return mockResponse({ ok: true });
}

/**
 * Termine l'intervention du héros. Réussie : l'incident est résolu. Échouée : il redevient
 * disponible pour un autre super-héros.
 * Future implémentation : `PUT ${API_URL}/hero/incidents/${id}/intervention` avec `{ outcome }`.
 */
export async function finishIntervention(
  token: string,
  id: string,
  outcome: InterventionOutcome,
): Promise<MutationResult> {
  const user = await userFromToken(token, "super-heros");
  const incident = findActionableIncident(id);
  if ("error" in incident) return mockResponse({ ok: false, error: incident.error });
  if (incident.status !== "en-cours" || incident.hero?.id !== user.id) {
    return mockResponse({ ok: false, error: "Vous n'intervenez pas sur cet incident." });
  }

  const now = new Date().toISOString();
  const intervention = interventionsMock.find(
    (existing) =>
      existing.incidentId === id && existing.hero.id === user.id && existing.outcome === null,
  );
  if (intervention) {
    intervention.endedAt = now;
    intervention.outcome = outcome;
  } else {
    // Héros affecté par la modération : aucune prise en charge enregistrée.
    interventionsMock.push({
      id: `intervention-${crypto.randomUUID()}`,
      incidentId: id,
      hero: incident.hero,
      startedAt: incident.reportedAt,
      endedAt: now,
      outcome,
    });
  }

  if (outcome === "reussi") {
    incident.status = "resolu";
    incident.resolvedAt = now;
  } else {
    incident.status = "signale";
    incident.hero = undefined;
  }

  for (const hero of [...superHeroesMock, ...superHeroProfilesMock]) {
    if (hero.id === user.id) hero.interventionsCount += 1;
  }
  const profile = superHeroProfilesMock.find(({ id: heroId }) => heroId === user.id);
  if (profile) profile.lastInterventionAt = now;

  return mockResponse({ ok: true });
}

/* ---------- Fiches publiques de l'encyclopédie ---------- */

/**
 * Interventions réussies d'un super-héros, des plus récentes aux plus anciennes.
 * Future implémentation : `GET ${API_URL}/heroes/${heroId}/incidents?status=resolu&limit=${limit}`
 */
export async function getResolvedIncidentsByHero(
  heroId: string,
  limit = 6,
): Promise<ResolvedIncident[]> {
  return mockResponse(
    incidentsMock
      .filter(isPublic)
      .filter(isResolved)
      .filter((incident) => incident.hero.id === heroId)
      .sort((a, b) => Date.parse(b.resolvedAt) - Date.parse(a.resolvedAt))
      .slice(0, limit),
  );
}

/**
 * Incidents attribués à un super-vilain, du plus récent au plus ancien.
 * Future implémentation : `GET ${API_URL}/villains/${villainId}/incidents?limit=${limit}`
 */
export async function getIncidentsByVillain(villainId: string, limit = 6): Promise<Incident[]> {
  return mockResponse(
    incidentsMock
      .filter(isPublic)
      .filter((incident) => incident.villain?.id === villainId)
      .sort(byMostRecent)
      .slice(0, limit),
  );
}
