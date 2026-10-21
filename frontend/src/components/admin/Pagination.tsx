import Link from "next/link";
import { pagePosition } from "@/lib/api";
import type { Page } from "@/types";

/** Paramètre d'URL `?page=` des listes paginées (à partir de 1). */
export type PageSearchParams = Promise<{ page?: string | string[] }>;

/** Numéro de page lu dans l'URL (`?page=` par défaut) : 1 si absent ou invalide. */
export async function readPageNumber(
  searchParams: Promise<Record<string, string | string[] | undefined>>,
  param = "page",
): Promise<number> {
  const raw = (await searchParams)[param];
  const value = Number(Array.isArray(raw) ? raw[0] : raw);
  return Number.isInteger(value) && value > 1 ? value : 1;
}

/**
 * Pages précédente et suivante d'une liste (liens : fonctionnent sans JavaScript et l'URL reste
 * partageable). Rien n'est affiché quand tout tient sur une page.
 */
export function Pagination({
  page,
  basePath,
  query = {},
  param = "page",
  anchor,
}: {
  page: Page<unknown>;
  basePath: string;
  /** Autres paramètres d'URL à conserver (filtres). */
  query?: Record<string, string | undefined>;
  /** Paramètre du numéro de page, pour distinguer plusieurs listes sur une même page. */
  param?: string;
  /** Ancre de la liste (sans `#`) : le changement de page y ramène au lieu du haut de page. */
  anchor?: string;
}) {
  const { current, count } = pagePosition(page);
  if (count <= 1) return null;

  const href = (target: number) => {
    const params = new URLSearchParams();
    for (const [name, value] of Object.entries(query)) if (value) params.set(name, value);
    if (target > 1) params.set(param, String(target));
    const search = params.toString();
    return `${search ? `${basePath}?${search}` : basePath}${anchor ? `#${anchor}` : ""}`;
  };
  const linkClasses =
    "inline-flex min-h-10 items-center rounded-full border border-border px-4 text-sm font-medium hover:border-muted";

  return (
    <nav aria-label="Pagination" className="mt-6 flex flex-wrap items-center gap-3">
      {current > 1 && (
        <Link href={href(current - 1)} className={linkClasses}>
          <span aria-hidden="true">←&nbsp;</span>Page précédente
        </Link>
      )}
      <p className="text-sm text-muted" aria-current="page">
        Page {current} sur {count}
      </p>
      {current < count && (
        <Link href={href(current + 1)} className={linkClasses}>
          Page suivante<span aria-hidden="true">&nbsp;→</span>
        </Link>
      )}
    </nav>
  );
}
