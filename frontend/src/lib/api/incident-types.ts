import { cache } from "react";
import { incidentTypesMock } from "@/mocks/incident-types";
import type {
  IncidentTypeCreationResult,
  IncidentTypeDefinition,
  IncidentTypeInput,
  Page,
  PageRequest,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { MAX_PAGE_SIZE, fromApiPage, pageQuery, toMockPage } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Référentiel des types d'incident (`incident_types`), géré par la modération. Les incidents
 * embarquent leur type (code et libellé) : la liste n'est utile qu'aux formulaires et au
 * back-office.
 */

function fromApiType(type: ApiSchema<"InterventionType">): IncidentTypeDefinition {
  return {
    id: type.id,
    code: type.code,
    label: type.label,
    endangersLife: type.endangersLife,
  };
}

const byLabel = (a: IncidentTypeDefinition, b: IncidentTypeDefinition) =>
  a.label.localeCompare(b.label, "fr");

/**
 * Tous les types proposés dans les formulaires, par libellé. Mémorisé le temps d'un rendu ou
 * d'une action (le formulaire et sa validation la relisent).
 * API : `GET /intervention-types`, page après page (100 types au plus par page).
 */
export const getIncidentTypes = cache(async (token: string): Promise<IncidentTypeDefinition[]> => {
  if (!isApiEnabled) {
    await userFromToken(token, ["citoyen", "super-heros", "admin"]);
    return mockResponse([...incidentTypesMock].sort(byLabel));
  }

  const types: IncidentTypeDefinition[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(
      `/intervention-types${pageQuery({ limit: MAX_PAGE_SIZE, offset })}`,
      { sessionToken: token },
    );
    const page = (await response.json()) as ApiSchema<"InterventionTypePage">;
    types.push(...page.items.map(fromApiType));
    if (page.items.length === 0 || types.length >= page.total) return types;
  }
});

/**
 * Référentiel complet pour le back-office, par libellé.
 * API : `GET /admin/intervention-types?limit=…&offset=…`.
 */
export async function getAllIncidentTypes(
  token: string,
  page: PageRequest = {},
): Promise<Page<IncidentTypeDefinition>> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    return mockResponse(toMockPage([...incidentTypesMock].sort(byLabel), page));
  }

  const response = await apiFetch(`/admin/intervention-types${pageQuery(page)}`, {
    sessionToken: token,
  });
  return fromApiPage((await response.json()) as ApiSchema<"InterventionTypePage">, fromApiType);
}

/**
 * Ajoute un type au référentiel.
 * API : `POST /admin/intervention-types` avec `{ code, label, endangersLife }`.
 */
export async function createIncidentType(
  token: string,
  input: IncidentTypeInput,
): Promise<IncidentTypeCreationResult> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    if (incidentTypesMock.some(({ code }) => code === input.code)) {
      return mockResponse({ ok: false, error: "code-already-used" });
    }
    const type = { id: Math.max(0, ...incidentTypesMock.map(({ id }) => id)) + 1, ...input };
    incidentTypesMock.push(type);
    return mockResponse({ ok: true, type });
  }

  try {
    const response = await apiFetch("/admin/intervention-types", {
      method: "POST",
      sessionToken: token,
      body: input satisfies ApiSchema<"InterventionTypePayload">,
    });
    return {
      ok: true,
      type: fromApiType((await response.json()) as ApiSchema<"InterventionType">),
    };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "incident_type_code_already_used") {
        return { ok: false, error: "code-already-used" };
      }
      if (error.code === "validation_failed") {
        return { ok: false, error: "invalid-fields", fields: error.fields };
      }
    }
    return toApiFailure(error, "Création d'un type d'incident");
  }
}
