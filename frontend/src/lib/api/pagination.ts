import type { Page, PageRequest } from "@/types";

/*
 * Listes paginées de l'API : `limit` (1 à 100, 20 par défaut) et `offset` en paramètres, réponse
 * `{ items, total, limit, offset }` (schémas `XxxPage` de l'OpenAPI).
 */

export const DEFAULT_PAGE_SIZE = 20;
export const MAX_PAGE_SIZE = 100;

/**
 * Chaîne de requête d'une liste : pagination et filtres (`undefined` et `""` sont ignorés).
 * Les bornes sont appliquées ici, l'API refusant toute valeur hors limites (422).
 *
 * `apiFetch(\`/admin/users${pageQuery(page, { status: "active" })}\`, …)`
 */
export function pageQuery(
  request: PageRequest = {},
  filters: Record<string, string | number | boolean | undefined> = {},
): string {
  const params = new URLSearchParams();
  for (const [name, value] of Object.entries(filters)) {
    if (value !== undefined && value !== "") params.set(name, String(value));
  }
  const { limit, offset } = normalizePage(request);
  params.set("limit", String(limit));
  params.set("offset", String(offset));
  return `?${params}`;
}

/** Convertit une page de l'API vers `Page<T>`, élément par élément (rôles, statuts, dates). */
export function fromApiPage<ApiItem, Item>(
  page: { items: ApiItem[]; total: number; limit: number; offset: number },
  convert: (item: ApiItem) => Item,
): Page<Item> {
  return {
    items: page.items.map(convert),
    total: page.total,
    limit: page.limit,
    offset: page.offset,
  };
}

/** Mock : découpe une liste complète comme le ferait l'API, pour garder la même signature. */
export function toMockPage<T>(items: readonly T[], request: PageRequest = {}): Page<T> {
  const { limit, offset } = normalizePage(request);
  return { items: items.slice(offset, offset + limit), total: items.length, limit, offset };
}

/** Numéro (à partir de 1) et nombre total de pages, pour un composant de pagination. */
export function pagePosition({ total, limit, offset }: Page<unknown>): {
  current: number;
  count: number;
} {
  return {
    current: Math.floor(offset / limit) + 1,
    count: Math.max(1, Math.ceil(total / limit)),
  };
}

/** Fenêtre dans les bornes de l'API : valeur absente ou invalide → défaut, hors bornes → borne. */
function normalizePage({ limit, offset }: PageRequest): Required<PageRequest> {
  return {
    limit:
      limit !== undefined && Number.isFinite(limit)
        ? Math.min(MAX_PAGE_SIZE, Math.max(1, Math.trunc(limit)))
        : DEFAULT_PAGE_SIZE,
    offset: offset !== undefined && Number.isFinite(offset) ? Math.max(0, Math.trunc(offset)) : 0,
  };
}
