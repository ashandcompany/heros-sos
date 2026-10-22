import Link from "next/link";
import { cn } from "@/components/ui/cn";
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
 * Pages précédente, suivante et numérotées d'une liste (liens : fonctionnent sans JavaScript et l'URL reste
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
    "inline-flex min-h-11 min-w-11 items-center justify-center border-2 border-border px-3 text-sm font-medium hover:border-muted";

  return (
    <nav aria-label="Pagination" className="mt-8 flex flex-wrap items-center gap-2">
      {current > 1 && (
        <Link href={href(current - 1)} className={linkClasses} rel="prev">
          <span aria-hidden="true">←&nbsp;</span>Précédente
        </Link>
      )}
      <ol className="flex flex-wrap items-center gap-2">
        {pageWindow(current, count).map((target, index) =>
          target === null ? (
            <li key={`gap-${index}`} aria-hidden="true" className="px-1 text-muted">
              …
            </li>
          ) : (
            <li key={target}>
              <Link
                href={href(target)}
                aria-label={`Page ${target} sur ${count}`}
                aria-current={target === current ? "page" : undefined}
                className={cn(
                  linkClasses,
                  target === current && "border-primary bg-primary/15 text-foreground",
                )}
              >
                {target}
              </Link>
            </li>
          ),
        )}
      </ol>
      {current < count && (
        <Link href={href(current + 1)} className={linkClasses} rel="next">
          Suivante<span aria-hidden="true">&nbsp;→</span>
        </Link>
      )}
    </nav>
  );
}

/** Pages affichées : la première, la dernière et deux voisines de la courante ; `null` = « … ». */
function pageWindow(current: number, count: number): (number | null)[] {
  const pages: (number | null)[] = [];
  for (let target = 1; target <= count; target++) {
    if (target === 1 || target === count || Math.abs(target - current) <= 2) {
      pages.push(target);
    } else if (pages.at(-1) !== null) {
      pages.push(null);
    }
  }
  return pages;
}
