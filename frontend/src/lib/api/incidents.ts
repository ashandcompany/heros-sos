import { formatLocation } from "@/lib/format";
import { geocodeLocation } from "@/lib/geocoding";
import { incidentsMock } from "@/mocks/incidents";
import { interventionsMock } from "@/mocks/interventions";
import { reviewsMock } from "@/mocks/reviews";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import type {
  CitizenIncidentDetail,
  GeoPoint,
  HeroIncident,
  Incident,
  IncidentDeclaration,
  IncidentMapData,
  IncidentPriority,
  IncidentStatus,
  IncidentUpdate,
  InterventionOutcome,
  InterventionRef,
  InterventionTeam,
  MapIncident,
  MapTeam,
  MapZone,
  ModerationStatus,
  MutationResult,
  OutcomeProposalResult,
  ResolvedIncident,
} from "@/types";
import { apiFailureMessage } from "@/lib/validation/form-state";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { getIncidentTypes } from "./incident-types";
import {
  getInterventionTeam,
  proposeOutcome,
  sessionUserId,
  toInterventionFailure,
} from "./interventions";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";
import { getVillain } from "./villains";

/*
 * Incidents et interventions. Branchés sur l'API : déclaration, « mes incidents », espace
 * super-héros (liste, fiche, carte, prise en charge, résultat, équipes : voir `interventions.ts`),
 * lecture par la modération, incidents en cours et carte publique (`GET /incidents` sans session),
 * fiche d'un incident pour un citoyen, historique d'un héros ou d'un vilain dans l'encyclopédie
 * (`GET /heroes/{heroId}/incidents`, `GET /villains/{id}/incidents`).
 *
 * Restent mockés, faute de route : derniers incidents résolus (la liste publique ne renvoie pas
 * les résolus), la priorité, la modération des
 * déclarations (validation, rejet, correction) et l'historique des interventions terminées.
 */

/** Fonctions que l'API ne couvre pas encore : les écrans les masquent quand elle est active. */
export const incidentFeatures = {
  /** Validation, rejet et correction d'une déclaration par la modération. */
  moderation: !isApiEnabled,
  /** Priorité fixée par les super-héros. */
  priority: !isApiEnabled,
  /** « Témoin » ou « victime » dans la déclaration. */
  reporterRole: !isApiEnabled,
  /** `GET /incidents` ne renvoie pas les incidents résolus aux héros ni aux modérateurs. */
  resolvedInLists: !isApiEnabled,
} as const;

/* ---------- Conversion depuis l'API ---------- */

type ApiIncident = ApiSchema<"Incident">;

/** Un incident en échec redevient disponible : l'écran le traite comme un nouveau signalement. */
const fromApiStatus: Record<ApiIncident["status"], IncidentStatus> = {
  reported: "signale",
  in_progress: "en-cours",
  succeeded: "resolu",
  failed: "signale",
};

const fromApiPriority: Record<ApiIncident["priority"], IncidentPriority> = {
  high: "haute",
  normal: "normale",
  low: "basse",
};

/**
 * Incident de l'API vers la forme du frontend. L'API diffuse un incident dès sa déclaration (aucune
 * vérification préalable) : `moderation` vaut donc `valide`. Le héros affiché est le lead de
 * l'équipe en cours, s'il est connu.
 */
function fromApiIncident(incident: ApiIncident, team: InterventionTeam | null = null): Incident {
  const { address, city, latitude, longitude } = incident.location;
  const villain = incident.villains?.[0];
  const lead = team?.members.find(({ role }) => role === "lead");
  return {
    id: incident.id,
    // Les incidents déclarés avant l'ajout du titre n'en ont pas.
    title: incident.title || [incident.typeLabel, city].filter(Boolean).join(" · "),
    type: { code: incident.typeCode ?? "", label: incident.typeLabel ?? "" },
    status: fromApiStatus[incident.status],
    moderation: "valide",
    priority: fromApiPriority[incident.priority],
    location: {
      city,
      address,
      coordinates:
        latitude != null && longitude != null ? { lat: latitude, lng: longitude } : undefined,
    },
    reportedAt: incident.createdAt,
    villain: villain ? { id: villain.id, alias: villain.alias } : null,
    hero: lead?.hero,
    description: incident.description,
    occurredAt: incident.occurredAt ?? undefined,
  };
}

function toApiHeroIncident(incident: ApiIncident, team: InterventionTeam | null): HeroIncident {
  const converted = fromApiIncident(incident, team);
  return { ...converted, priority: converted.priority ?? "normale", interventions: [], team };
}

/** Incidents encore à traiter : un incident en échec redevient disponible. */
const UNRESOLVED_STATUSES: readonly ApiIncident["status"][] = ["reported", "in_progress", "failed"];

/**
 * Toutes les pages de `GET /incidents` (100 par appel), du plus récent au plus ancien. L'API
 * renvoie tous les statuts : par défaut, seuls les incidents non résolus sont demandés
 * (`statuses`, vide pour tout lire). Avec la session d'un citoyen, l'API ne renvoie que ses
 * propres déclarations : les écrans publics appellent donc sans jeton.
 */
async function fetchAllIncidents(
  token: string | undefined,
  filters: Record<string, string> = {},
  statuses: readonly ApiIncident["status"][] = UNRESOLVED_STATUSES,
): Promise<ApiIncident[]> {
  const statusQuery = statuses.map((status) => `&status=${status}`).join("");
  const incidents: ApiIncident[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(
      `/incidents${pageQuery({ limit: MAX_PAGE_SIZE, offset }, filters)}${statusQuery}`,
      { sessionToken: token },
    );
    const page = (await response.json()) as ApiSchema<"IncidentPage">;
    incidents.push(...page.items);
    if (page.items.length === 0 || incidents.length >= page.total) return incidents;
  }
}

/** `GET /incidents/{id}` (public), `null` si l'incident n'existe pas. */
async function fetchIncident(token: string, id: string): Promise<ApiIncident | null> {
  try {
    const response = await apiFetch(`/incidents/${encodeURIComponent(id)}`, {
      sessionToken: token,
    });
    return (await response.json()) as ApiIncident;
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}

/** Incidents et leur équipe en cours (`GET /interventions/{id}`, en parallèle). */
async function withTeams(
  token: string,
  incidents: ApiIncident[],
  meId: string | null,
): Promise<HeroIncident[]> {
  return Promise.all(
    incidents.map(async (incident) =>
      toApiHeroIncident(
        incident,
        incident.interventionId
          ? await getInterventionTeam(token, incident.interventionId, meId)
          : null,
      ),
    ),
  );
}

const featureUnavailable: MutationResult = {
  ok: false,
  error: "Cette fonction n'est pas encore disponible.",
};

/** Type du référentiel (API ou mock), réduit à ce qu'embarque un incident ; `undefined` si inconnu. */
async function findIncidentType(
  token: string,
  typeId: number,
): Promise<Incident["type"] | undefined> {
  const type = (await getIncidentTypes(token)).find(({ id }) => id === typeId);
  return type && { code: type.code, label: type.label };
}

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
 * API : `GET /incidents` sans session, toutes les pages ; l'API ne filtre pas par ville, le
 * filtre est appliqué ici.
 */
export async function getOngoingIncidents(city?: string): Promise<Incident[]> {
  const wanted = city ? normalizeCity(city) : null;
  const inCity = (incident: Incident) =>
    !wanted || normalizeCity(incident.location.city) === wanted;

  if (isApiEnabled) {
    return (await fetchAllIncidents(undefined))
      .map((incident) => fromApiIncident(incident))
      .filter(inCity);
  }

  const ongoing = incidentsMock
    .filter(isPublic)
    .filter((incident) => incident.status !== "resolu")
    .filter(inCity)
    .sort(byMostRecent);

  return mockResponse(ongoing);
}

/**
 * Incidents déclarés par l'utilisateur connecté (citoyen ou super-héros), du plus récent au plus
 * ancien (y compris ceux rejetés par la modération, pour qu'il en soit informé).
 * API : `GET /incidents?reporterId={moi}`, toutes les pages. Le héros qui intervient n'est pas
 * connu : un citoyen n'a pas accès aux interventions.
 */
export async function getMyIncidents(token: string): Promise<Incident[]> {
  if (isApiEnabled) {
    const meId = await sessionUserId(token);
    if (!meId) return [];
    return (await fetchAllIncidents(token, { reporterId: meId }, [])).map((incident) =>
      fromApiIncident(incident),
    );
  }

  const user = await userFromToken(token, ["citoyen", "super-heros"]);
  return mockResponse(
    incidentsMock.filter((incident) => incident.reporterId === user.id).sort(byMostRecent),
  );
}

/**
 * Déclare un incident ; il est diffusé aux super-héros.
 * Un citoyen, un super-héros ou un modérateur peut déclarer. Mock : la déclaration d'un modérateur
 * est validée d'office, celle d'un citoyen ou d'un super-héros attend la vérification de la
 * modération. L'API, elle, prévient aussitôt les héros et les modérateurs.
 * API : `POST /incidents`, avec la position géocodée par le frontend (carte des héros).
 */
export async function declareIncident(
  token: string,
  declaration: IncidentDeclaration,
): Promise<MutationResult> {
  const coordinates = (await geocodeLocation(declaration.location)) ?? undefined;

  if (isApiEnabled) {
    try {
      await apiFetch("/incidents", {
        method: "POST",
        sessionToken: token,
        body: {
          ...(declaration.id ? { id: declaration.id } : {}),
          typeId: declaration.typeId,
          title: declaration.title,
          description: declaration.description,
          // Le formulaire ne le demande pas : nombre connu après l'intervention.
          citizensRescued: 0,
          location: {
            address: declaration.location.address,
            city: declaration.location.city,
            latitude: coordinates?.lat ?? null,
            longitude: coordinates?.lng ?? null,
          },
          occurredAt: declaration.occurredAt,
          villainIds: declaration.villainId ? [declaration.villainId] : [],
        } satisfies ApiSchema<"IncidentPayload">,
      });
      return { ok: true };
    } catch (error) {
      if (error instanceof ApiError && error.code === "validation_failed") {
        return {
          ok: false,
          error: "Certaines informations sont invalides. Vérifiez le formulaire.",
        };
      }
      return { ok: false, error: apiFailureMessage(toApiFailure(error, "Déclaration d'incident")) };
    }
  }

  const user = await userFromToken(token, ["citoyen", "super-heros", "admin"]);
  const villain = declaration.villainId ? await getVillain(declaration.villainId) : null;
  const type = await findIncidentType(token, declaration.typeId);
  if (!type) return mockResponse({ ok: false, error: "Type d'incident inconnu." });

  incidentsMock.push({
    id: `incident-${crypto.randomUUID()}`,
    title: declaration.title,
    type,
    status: "signale",
    moderation: user.role === "admin" ? "valide" : "en-attente",
    location: { ...declaration.location, coordinates },
    reportedAt: new Date().toISOString(),
    occurredAt: declaration.occurredAt ?? undefined,
    villain: villain ? { id: villain.id, alias: villain.alias } : null,
    description: declaration.description,
    reporterId: user.id,
    reporterRole: declaration.reporterRole ?? undefined,
  });

  return mockResponse({ ok: true });
}

/* ---------- Back-office (rôle admin) ---------- */

/**
 * Tous les incidents, éventuellement filtrés par décision de modération,
 * du plus récent au plus ancien.
 * API : `GET /incidents` (incidents signalés, en cours ou en échec, sans les résolus), avec leur
 * équipe en cours. L'API n'a pas de modération des déclarations : `moderation` est ignoré.
 */
export async function getAllIncidents(
  token: string,
  moderation?: ModerationStatus,
): Promise<Incident[]> {
  if (isApiEnabled) return withTeams(token, await fetchAllIncidents(token), null);

  await userFromToken(token, "admin");
  return mockResponse(
    incidentsMock
      .filter((incident) => !moderation || incident.moderation === moderation)
      .sort(byMostRecent),
  );
}

/**
 * Un incident et son équipe en cours, pour le back-office.
 * API : `GET /incidents/{id}` puis `GET /interventions/{interventionId}` (404 → `null`).
 */
export async function getIncident(token: string, id: string): Promise<HeroIncident | null> {
  if (isApiEnabled) {
    const incident = await fetchIncident(token, id);
    return incident ? (await withTeams(token, [incident], null))[0] : null;
  }

  await userFromToken(token, "admin");
  const incident = incidentsMock.find((existing) => existing.id === id);
  return mockResponse(incident ? toHeroIncident(incident, null) : null);
}

/**
 * Corrige les informations d'un incident (titre, lieu, vilain, héros, statut…).
 * Future implémentation : `PATCH ${API_URL}/admin/incidents/${id}` avec l'`IncidentUpdate`
 * (absente du contrat : voir `incidentFeatures.moderation`).
 */
export async function updateIncident(
  token: string,
  id: string,
  update: IncidentUpdate,
): Promise<MutationResult> {
  if (isApiEnabled) return featureUnavailable;
  await userFromToken(token, "admin");
  const incident = incidentsMock.find((existing) => existing.id === id);
  if (!incident) return mockResponse({ ok: false, error: "Incident introuvable." });

  const villain = update.villainId ? await getVillain(update.villainId) : null;
  const hero = superHeroesMock.find(({ id: heroId }) => heroId === update.heroId);
  if (update.status !== "signale" && !hero) {
    return mockResponse({
      ok: false,
      error: "Un incident pris en charge ou résolu doit avoir un super-héros.",
    });
  }

  const type = await findIncidentType(token, update.typeId);
  if (!type) return mockResponse({ ok: false, error: "Type d'incident inconnu." });

  incident.title = update.title;
  incident.type = type;
  incident.description = update.description;
  // Le lieu change : on le géocode de nouveau (sinon on garde la position connue).
  const moved =
    update.location.address !== (incident.location.address ?? "") ||
    normalizeCity(update.location.city) !== normalizeCity(incident.location.city);
  const coordinates = moved
    ? ((await geocodeLocation(update.location)) ?? undefined)
    : incident.location.coordinates;
  incident.location = { ...incident.location, ...update.location, coordinates };
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
 * Future implémentation : `PUT ${API_URL}/admin/incidents/${id}/moderation` avec `{ decision }`
 * (absente du contrat : voir `incidentFeatures.moderation`).
 */
export async function moderateIncident(
  token: string,
  id: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  if (isApiEnabled) return featureUnavailable;
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

/**
 * Mock : un seul héros intervient à la fois, l'équipe se réduit à lui. `meId` reconnaît le héros
 * connecté (`null` pour un modérateur).
 */
function mockTeam(incident: Incident, meId: string | null): InterventionTeam | null {
  if (incident.status !== "en-cours" || !incident.hero) return null;
  const open = interventionsMock.find(
    ({ incidentId, outcome }) => incidentId === incident.id && outcome === null,
  );
  return {
    id: open?.id ?? `intervention-${incident.id}`,
    startedAt: open?.startedAt ?? incident.reportedAt,
    members: [
      {
        hero: incident.hero,
        role: "lead",
        participation: "confirme",
        vote: "en-attente",
        isMe: incident.hero.id === meId,
      },
    ],
    proposedOutcome: null,
    proposedBy: null,
    notes: "",
  };
}

function toHeroIncident(incident: Incident, meId: string | null): HeroIncident {
  return {
    ...incident,
    priority: incident.priority ?? "normale",
    interventions: interventionsMock
      .filter(({ incidentId, outcome }) => incidentId === incident.id && outcome !== null)
      .sort((a, b) => Date.parse(b.startedAt) - Date.parse(a.startedAt)),
    team: mockTeam(incident, meId),
  };
}

const byHeroQueue = (a: HeroIncident, b: HeroIncident) =>
  statusRank[a.status] - statusRank[b.status] ||
  priorityRank[a.priority] - priorityRank[b.priority] ||
  byMostRecent(a, b);

/**
 * Signalements diffusés aux super-héros, avec leur équipe en cours, triés par statut (à prendre
 * en charge d'abord), puis par priorité et du plus récent au plus ancien.
 * API : `GET /incidents` (sans les incidents résolus), puis l'équipe de chaque intervention en cours.
 */
export async function getHeroIncidents(
  token: string,
  statuses?: readonly IncidentStatus[],
): Promise<HeroIncident[]> {
  const wanted = (status: IncidentStatus) => !statuses || statuses.includes(status);

  if (isApiEnabled) {
    const [incidents, meId] = await Promise.all([fetchAllIncidents(token), sessionUserId(token)]);
    const kept = incidents.filter(({ status }) => wanted(fromApiStatus[status]));
    return (await withTeams(token, kept, meId)).sort(byHeroQueue);
  }

  const user = await userFromToken(token, "super-heros");
  const incidents = incidentsMock
    .filter(isPublic)
    .filter(({ status }) => wanted(status))
    .map((incident) => toHeroIncident(incident, user.id))
    .sort(byHeroQueue);

  return mockResponse(incidents);
}

/** Fiche d'un incident pour un héros. API : `GET /incidents/{id}` et son équipe (404 → `null`). */
export async function getHeroIncident(token: string, id: string): Promise<HeroIncident | null> {
  if (isApiEnabled) {
    const [incident, meId] = await Promise.all([fetchIncident(token, id), sessionUserId(token)]);
    return incident ? (await withTeams(token, [incident], meId))[0] : null;
  }

  const user = await userFromToken(token, "super-heros");
  const incident = incidentsMock.find((existing) => existing.id === id && isPublic(existing));
  return mockResponse(incident ? toHeroIncident(incident, user.id) : null);
}

/** Rayon minimal d'une zone touchée, pour qu'un incident isolé reste visible. */
const ZONE_MIN_RADIUS_M = 800;
/** Marge autour de l'incident le plus éloigné du centre de la zone. */
const ZONE_MARGIN_M = 400;

/** Distance en mètres entre deux points (formule de haversine). */
function distanceInMeters(a: GeoPoint, b: GeoPoint): number {
  const toRad = (degrees: number) => (degrees * Math.PI) / 180;
  const h =
    Math.sin(toRad(b.lat - a.lat) / 2) ** 2 +
    Math.cos(toRad(a.lat)) * Math.cos(toRad(b.lat)) * Math.sin(toRad(b.lng - a.lng) / 2) ** 2;
  return 2 * 6_371_000 * Math.asin(Math.sqrt(h));
}

function toZone(city: string, incidents: MapIncident[]): MapZone {
  const center = {
    lat: incidents.reduce((sum, { position }) => sum + position.lat, 0) / incidents.length,
    lng: incidents.reduce((sum, { position }) => sum + position.lng, 0) / incidents.length,
  };
  const farthest = Math.max(...incidents.map(({ position }) => distanceInMeters(center, position)));
  return {
    city,
    center,
    radius: Math.max(ZONE_MIN_RADIUS_M, farthest + ZONE_MARGIN_M),
    incidentsCount: incidents.length,
  };
}

/** Deux décimales ≈ 1 km : assez pour situer le quartier, pas pour retrouver une adresse. */
const roundCoordinate = (value: number) => Math.round(value * 100) / 100;

/**
 * Données d'une carte, à partir des incidents non résolus. `hero` : positions exactes, adresse et
 * héros en intervention. `public` : positions arrondies, ville et quartier seulement, aucun héros.
 * Seuls les champs nécessaires à l'affichage sont renvoyés (ils partent vers le navigateur).
 */
function buildIncidentMap(
  candidates: Array<Incident & { team?: InterventionTeam | null }>,
  audience: "hero" | "public",
): IncidentMapData {
  const open = candidates
    .filter((incident) => incident.status !== "resolu")
    .sort(
      (a, b) =>
        priorityRank[a.priority ?? "normale"] - priorityRank[b.priority ?? "normale"] ||
        byMostRecent(a, b),
    );

  const incidents: MapIncident[] = [];
  const teams: MapTeam[] = [];
  const byCity = new Map<string, { city: string; incidents: MapIncident[] }>();

  for (const incident of open) {
    const exact = incident.location.coordinates;
    if (!exact || incident.status === "resolu") continue;
    const position =
      audience === "public"
        ? { lat: roundCoordinate(exact.lat), lng: roundCoordinate(exact.lng) }
        : exact;

    const mapIncident: MapIncident = {
      id: incident.id,
      title: incident.title,
      type: incident.type,
      status: incident.status,
      priority: incident.priority ?? "normale",
      place:
        audience === "public"
          ? formatLocation(incident.location)
          : [incident.location.address, formatLocation(incident.location)]
              .filter(Boolean)
              .join(", "),
      position,
      reportedAt: incident.reportedAt,
    };
    incidents.push(mapIncident);

    const key = normalizeCity(incident.location.city);
    const zone = byCity.get(key) ?? { city: incident.location.city, incidents: [] };
    zone.incidents.push(mapIncident);
    byCity.set(key, zone);

    const lead = incident.team?.members.find(({ role }) => role === "lead");
    if (audience === "hero" && incident.status === "en-cours" && incident.team && lead) {
      teams.push({
        hero: lead.hero,
        isMe: incident.team.members.some(
          ({ isMe, participation }) => isMe && participation === "confirme",
        ),
        incidentId: incident.id,
        position,
        startedAt: incident.team.startedAt,
      });
    }
  }

  return {
    incidents,
    teams,
    zones: Array.from(byCity.values(), ({ city, incidents: zone }) => toZone(city, zone)),
    unlocatedCount: open.length - incidents.length,
  };
}

/**
 * Carte opérationnelle : incidents non résolus, super-héros en intervention et zones touchées.
 * API : construite à partir de `getHeroIncidents` (position enregistrée à la déclaration).
 */
export async function getIncidentMap(token: string): Promise<IncidentMapData> {
  return buildIncidentMap(await getHeroIncidents(token), "hero");
}

/**
 * Carte publique, visible sans compte : incidents non résolus à une position approximative, sans
 * les super-héros. Mock : seulement ceux validés par la modération (un signalement en attente
 * peut être faux) ; l'API n'a pas de modération des déclarations.
 * API : `GET /incidents` sans session, toutes les pages.
 */
export async function getPublicIncidentMap(): Promise<IncidentMapData> {
  if (isApiEnabled) {
    const incidents = await fetchAllIncidents(undefined);
    return buildIncidentMap(
      incidents.map((incident) => fromApiIncident(incident)),
      "public",
    );
  }

  return mockResponse(
    buildIncidentMap(
      incidentsMock.filter((incident) => incident.moderation === "valide"),
      "public",
    ),
  );
}

/** Incident diffusé et non résolu, sur lequel un héros peut agir. */
function findActionableIncident(id: string): Incident | { error: string } {
  const incident = incidentsMock.find((existing) => existing.id === id && isPublic(existing));
  if (!incident) return { error: "Incident introuvable." };
  if (incident.status === "resolu") return { error: "Cet incident est déjà résolu." };
  return incident;
}

/**
 * Future implémentation : `PUT ${API_URL}/hero/incidents/${id}/priority` avec `{ priority }`
 * (absente du contrat : voir `incidentFeatures.priority`).
 */
export async function setIncidentPriority(
  token: string,
  id: string,
  priority: IncidentPriority,
): Promise<MutationResult> {
  if (isApiEnabled) return featureUnavailable;
  await userFromToken(token, "super-heros");
  const incident = findActionableIncident(id);
  if ("error" in incident) return mockResponse({ ok: false, error: incident.error });

  incident.priority = priority;
  return mockResponse({ ok: true });
}

/**
 * Prise en charge d'un incident : le héros démarre l'intervention et en devient le lead. Une seule
 * intervention à la fois par incident (409 si un autre héros l'a démarrée entre-temps) ; les
 * autres héros rejoignent ensuite son équipe.
 * API : `POST /incidents/{incidentId}/interventions`.
 */
export async function takeChargeOfIncident(token: string, id: string): Promise<MutationResult> {
  if (isApiEnabled) {
    try {
      await apiFetch(`/incidents/${encodeURIComponent(id)}/interventions`, {
        method: "POST",
        sessionToken: token,
      });
      return { ok: true };
    } catch (error) {
      return toInterventionFailure(error, "Prise en charge d'un incident", {
        intervention_already_active:
          "Un autre super-héros intervient déjà : rejoignez son équipe depuis la fiche de l'incident.",
        hero_already_in_intervention: "Vous participez déjà à une autre intervention en cours.",
      });
    }
  }

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
 * Déclare le résultat de l'intervention. Héros seul : elle est close aussitôt (réussie : l'incident
 * est résolu ; échouée : il redevient disponible). Équipe : le résultat est soumis au vote des
 * membres confirmés (`closed: false`).
 * API : `POST /interventions/{id}/outcome` (voir `proposeOutcome`).
 */
export async function finishIntervention(
  token: string,
  ref: InterventionRef,
  outcome: InterventionOutcome,
  notes = "",
): Promise<OutcomeProposalResult> {
  if (isApiEnabled) return proposeOutcome(token, ref, outcome, notes);

  const id = ref.incidentId;
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

  return mockResponse({ ok: true, closed: true });
}

/* ---------- Fiches publiques de l'encyclopédie ---------- */

type ApiIncidentSummary = ApiSchema<"IncidentSummary">;

/**
 * Incident de l'historique public (sans description ni adresse précise). Le héros affiché est le
 * lead de la dernière intervention ; le vilain, le premier associé.
 */
function fromApiSummary(incident: ApiIncidentSummary): Incident {
  const villain = incident.villains[0];
  return {
    id: incident.id,
    title: incident.title || [incident.typeLabel, incident.city].filter(Boolean).join(" · "),
    type: { code: incident.typeCode, label: incident.typeLabel },
    status: fromApiStatus[incident.status],
    moderation: "valide",
    location: { city: incident.city },
    reportedAt: incident.createdAt,
    resolvedAt: incident.resolvedAt ?? undefined,
    villain: villain ? { id: villain.id, alias: villain.alias } : null,
    hero: incident.lead ? { id: incident.lead.id, alias: incident.lead.alias } : undefined,
  };
}

/** Une page d'historique public ; vide si la fiche n'existe pas (404) ou si l'identifiant est invalide. */
async function fetchIncidentSummaries(path: string, limit: number): Promise<Incident[]> {
  try {
    const response = await apiFetch(`${path}${pageQuery({ limit, offset: 0 })}`);
    const page = (await response.json()) as ApiSchema<"IncidentSummaryPage">;
    return page.items.map(fromApiSummary);
  } catch (error) {
    if (error instanceof ApiError && (error.status === 404 || error.status === 422)) return [];
    throw error;
  }
}

/**
 * Interventions réussies d'un super-héros, des plus récentes aux plus anciennes.
 * API : `GET /heroes/{heroId}/incidents` (publique).
 */
export async function getResolvedIncidentsByHero(
  heroId: string,
  limit = 6,
): Promise<ResolvedIncident[]> {
  if (isApiEnabled) {
    const incidents = await fetchIncidentSummaries(
      `/heroes/${encodeURIComponent(heroId)}/incidents`,
      limit,
    );
    return incidents.filter(isResolved);
  }
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
 * API : `GET /villains/{id}/incidents` (publique).
 */
export async function getIncidentsByVillain(villainId: string, limit = 6): Promise<Incident[]> {
  if (isApiEnabled) {
    return fetchIncidentSummaries(`/villains/${encodeURIComponent(villainId)}/incidents`, limit);
  }
  return mockResponse(
    incidentsMock
      .filter(isPublic)
      .filter((incident) => incident.villain?.id === villainId)
      .sort(byMostRecent)
      .slice(0, limit),
  );
}

/* ---------- Détail d'un incident (espace citoyen) ---------- */

/**
 * Un citoyen peut consulter ses propres déclarations (y compris rejetées, pour en être informé)
 * et les incidents diffusés publiquement. Pour ces derniers, l'identité du déclarant est retirée.
 * API : `GET /incidents/{id}` (public, 404 → `null`), sans le suivi des interventions (fermées
 * aux citoyens). L'incident ne dit pas qui l'a déclaré : `isMine` vient de
 * `GET /incidents?reporterId={moi}`, lu en parallèle.
 */
export async function getCitizenIncident(
  token: string,
  id: string,
): Promise<CitizenIncidentDetail | null> {
  if (isApiEnabled) {
    const [incident, mine] = await Promise.all([fetchIncident(token, id), getMyIncidents(token)]);
    if (!incident) return null;
    return {
      incident: fromApiIncident(incident),
      isMine: mine.some((own) => own.id === id),
      interventions: [],
      review: null,
    };
  }

  const user = await userFromToken(token, "citoyen");
  const incident = incidentsMock.find((existing) => existing.id === id);
  if (!incident) return mockResponse(null);

  const isMine = incident.reporterId === user.id;
  if (!isMine && !isPublic(incident)) return mockResponse(null);

  return mockResponse({
    incident: isMine ? incident : { ...incident, reporterId: undefined },
    isMine,
    interventions: interventionsMock
      .filter(({ incidentId }) => incidentId === id)
      .sort((a, b) => Date.parse(a.startedAt) - Date.parse(b.startedAt)),
    review: isMine ? (reviewsMock.find(({ incidentId }) => incidentId === id) ?? null) : null,
  });
}

/**
 * Super-héros qu'on peut affecter à un incident mocké (correction par la modération), triés par
 * pseudo : ceux des mocks, que les incidents mockés référencent. La correction n'existe pas avec
 * l'API (`incidentFeatures.moderation`) : à supprimer avec elle.
 */
export async function getIncidentHeroes(): Promise<Array<{ id: string; alias: string }>> {
  return mockResponse(
    superHeroesMock
      .map(({ id, alias }) => ({ id, alias }))
      .sort((a, b) => a.alias.localeCompare(b.alias, "fr")),
  );
}
