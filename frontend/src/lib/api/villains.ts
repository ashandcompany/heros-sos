import { cache } from "react";
import { villainsMock } from "@/mocks/villains";
import type { Villain } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";

/*
 * Super-vilains : lecture publique (encyclopédie, formulaires de déclaration). Les champs d'état
 * civil, le secteur et les spécialités (`typeIds`) du contrat ne sont pas encore affichés.
 */

function fromApiVillain(villain: ApiSchema<"Villain">): Villain {
  return {
    id: villain.id,
    alias: villain.alias,
    threatLevel: villain.threatLevel,
    description: villain.description ?? "",
    reportedIncidentsCount: villain.reportedIncidentsCount,
    status: villain.status,
  };
}

const byAlias = (a: Villain, b: Villain) => a.alias.localeCompare(b.alias, "fr");

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
