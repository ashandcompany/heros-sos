import { cache } from "react";
import { incidentTypesMock } from "@/mocks/incident-types";
import { villainsMock } from "@/mocks/villains";
import type {
  Page,
  PageRequest,
  Villain,
  VillainInput,
  VillainSaveResult,
  VillainSpecialty,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { MAX_PAGE_SIZE, fromApiPage, pageQuery, toMockPage } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Super-vilains. Lecture publique (encyclopédie, formulaires de déclaration) ; création et
 * modification par la modération (back-office). L'API laisse aussi tout utilisateur connecté créer
 * un vilain : ce n'est pas encore proposé dans les formulaires.
 */

function fromApiVillain(villain: ApiSchema<"Villain">): Villain {
  return {
    id: villain.id,
    alias: villain.alias,
    threatLevel: villain.threatLevel,
    description: villain.description ?? "",
    reportedIncidentsCount: villain.reportedIncidentsCount,
    status: villain.status,
    lastName: villain.lastName ?? "",
    firstName: villain.firstName ?? "",
    mainSector: villain.mainSector ?? "",
    typeIds: villain.typeIds,
    specialties: villain.specialties.map(({ id, code, label }) => ({ id, code, label })),
  };
}

const byAlias = (a: Villain, b: Villain) => a.alias.localeCompare(b.alias, "fr");

/** Mock : spécialités détaillées à partir du référentiel mock, par libellé (comme l'API). */
function mockSpecialties(typeIds: number[]): VillainSpecialty[] {
  return incidentTypesMock
    .filter(({ id }) => typeIds.includes(id))
    .map(({ id, code, label }) => ({ id, code, label }))
    .sort((a, b) => a.label.localeCompare(b.label, "fr"));
}

/** Recherche du back-office : début d'alias, sans tenir compte de la casse ni des accents. */
const normalizeAlias = (alias: string) =>
  alias
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .trim()
    .toLowerCase();

/**
 * Tous les super-vilains connus, triés par nom. Mémorisé le temps d'un rendu ou d'une action.
 * API : `GET /villains`, page après page (100 vilains au plus par page).
 */
export const getVillains = cache(async (): Promise<Villain[]> => {
  if (!isApiEnabled) return mockResponse([...villainsMock].sort(byAlias));

  const villains: Villain[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(`/villains${pageQuery({ limit: MAX_PAGE_SIZE, offset })}`);
    const page = (await response.json()) as ApiSchema<"VillainPage">;
    villains.push(...page.items.map(fromApiVillain));
    if (page.items.length === 0 || villains.length >= page.total) return villains.sort(byAlias);
  }
});

/**
 * Une page de vilains pour le back-office, par alias, éventuellement filtrée par début d'alias.
 * API : `GET /villains?q=…&limit=…&offset=…`.
 */
export async function getVillainsPage(page: PageRequest = {}, search = ""): Promise<Page<Villain>> {
  if (!isApiEnabled) {
    const prefix = normalizeAlias(search);
    const matching = villainsMock
      .filter(({ alias }) => normalizeAlias(alias).startsWith(prefix))
      .sort(byAlias);
    return mockResponse(toMockPage(matching, page));
  }

  const response = await apiFetch(`/villains${pageQuery(page, { q: search.trim() })}`);
  return fromApiPage((await response.json()) as ApiSchema<"VillainPage">, fromApiVillain);
}

/**
 * Fiche d'un super-vilain, `null` s'il n'existe pas. Mémorisée le temps d'un rendu (métadonnées
 * et page la relisent).
 * API : `GET /villains/{id}` (404, y compris pour un identifiant mal formé → `null`).
 */
export const getVillain = cache(async (id: string): Promise<Villain | null> => {
  if (!isApiEnabled) return mockResponse(villainsMock.find((villain) => villain.id === id) ?? null);

  try {
    const response = await apiFetch(`/villains/${encodeURIComponent(id)}`);
    return fromApiVillain((await response.json()) as ApiSchema<"Villain">);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
});

/**
 * Crée une fiche de vilain.
 * API : `POST /villains`, qui crée toujours le vilain en liberté avec une menace faible ; si la
 * saisie en diffère, un `PATCH /villains/{id}` (réservé aux modérateurs) suit aussitôt. S'il
 * échoue, le vilain existe déjà : on le renvoie tel que créé, sa fiche affichera ses vraies valeurs.
 * L'alias est unique à la casse près, comme la contrainte de la base.
 */
export async function createVillain(
  token: string,
  input: VillainInput,
): Promise<VillainSaveResult> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    if (villainsMock.some(({ alias }) => alias === input.alias)) {
      return mockResponse({ ok: false, error: "alias-already-used" });
    }
    const villain: Villain = {
      id: `villain-${crypto.randomUUID()}`,
      reportedIncidentsCount: 0,
      ...input,
      specialties: mockSpecialties(input.typeIds),
    };
    villainsMock.push(villain);
    return mockResponse({ ok: true, villain });
  }

  try {
    const response = await apiFetch("/villains", {
      method: "POST",
      sessionToken: token,
      body: {
        alias: input.alias,
        lastName: input.lastName,
        firstName: input.firstName,
        mainSector: input.mainSector,
        description: input.description,
        typeIds: input.typeIds,
      } satisfies ApiSchema<"VillainPayload">,
    });
    const created = fromApiVillain((await response.json()) as ApiSchema<"Villain">);
    if (created.threatLevel === input.threatLevel && created.status === input.status) {
      return { ok: true, villain: created };
    }
    const updated = await updateVillain(token, created.id, input);
    return updated.ok ? updated : { ok: true, villain: created };
  } catch (error) {
    return toSaveFailure(error, "Création d'un super-vilain");
  }
}

/**
 * Remplace une fiche de vilain, spécialités, menace et statut compris.
 * API : `PATCH /villains/{id}` avec le `VillainUpdatePayload` complet.
 */
export async function updateVillain(
  token: string,
  id: string,
  input: VillainInput,
): Promise<VillainSaveResult> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const villain = villainsMock.find((existing) => existing.id === id);
    if (!villain) return mockResponse({ ok: false, error: "not-found" });
    if (villainsMock.some((other) => other.id !== id && other.alias === input.alias))
      return mockResponse({ ok: false, error: "alias-already-used" });
    Object.assign(villain, input, { specialties: mockSpecialties(input.typeIds) });
    return mockResponse({ ok: true, villain });
  }

  try {
    const response = await apiFetch(`/villains/${encodeURIComponent(id)}`, {
      method: "PATCH",
      sessionToken: token,
      body: {
        alias: input.alias,
        // Une chaîne vide efface le champ ; `null` le laisserait inchangé.
        lastName: input.lastName,
        firstName: input.firstName,
        mainSector: input.mainSector,
        description: input.description,
        typeIds: input.typeIds,
        threatLevel: input.threatLevel,
        status: input.status,
      } satisfies ApiSchema<"VillainUpdatePayload">,
    });
    return { ok: true, villain: fromApiVillain((await response.json()) as ApiSchema<"Villain">) };
  } catch (error) {
    return toSaveFailure(error, "Modification d'un super-vilain");
  }
}

function toSaveFailure(error: unknown, context: string): VillainSaveResult {
  if (error instanceof ApiError) {
    if (error.code === "villain_alias_already_used") {
      return { ok: false, error: "alias-already-used" };
    }
    if (error.status === 404) return { ok: false, error: "not-found" };
    if (error.code === "validation_failed") {
      return { ok: false, error: "invalid-fields", fields: error.fields };
    }
  }
  return toApiFailure(error, context);
}
