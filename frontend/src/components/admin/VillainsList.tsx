import Link from "next/link";
import { threatTone } from "@/components/encyclopedia/VillainCard";
import { Badge } from "@/components/ui/Badge";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { getVillainsPage } from "@/lib/api";
import { formatInteger, threatLevelLabels, villainStatusLabels } from "@/lib/format";
import { adminVillainRoute, routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import { EmptyState } from "./ModerationFilter";
import { Pagination, readPageNumber } from "./Pagination";

const PAGE_SIZE = 50;

export type VillainsSearchParams = Promise<{ page?: string | string[]; q?: string | string[] }>;

/** Fiches des super-vilains. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function VillainsList({ searchParams }: { searchParams: VillainsSearchParams }) {
  await requireAdminSession();
  const { q } = await searchParams;
  const search = (Array.isArray(q) ? q[0] : q)?.trim() ?? "";
  const pageNumber = await readPageNumber(searchParams);
  const page = await getVillainsPage(
    { limit: PAGE_SIZE, offset: (pageNumber - 1) * PAGE_SIZE },
    search,
  );

  return (
    <>
      <form
        role="search"
        action={routes.adminVillains}
        className="mb-4 flex flex-wrap items-end gap-2"
      >
        <label className="text-sm font-medium">
          <span className="mb-2 block">Rechercher par alias</span>
          <input
            type="search"
            name="q"
            defaultValue={search}
            placeholder="Début de l'alias"
            className="min-h-10 w-64 max-w-full border-2 border-border bg-background px-3 text-foreground placeholder:text-muted"
          />
        </label>
        <button type="submit" className={buttonClasses("secondary")}>
          Rechercher
        </button>
      </form>

      {page.total === 0 ? (
        <EmptyState>
          {search
            ? `Aucun super-vilain dont l'alias commence par « ${search} ».`
            : "Aucun super-vilain répertorié pour l'instant. Créez la première fiche ci-dessus."}
        </EmptyState>
      ) : (
        <>
          <p className="mb-3 text-sm text-muted">
            {formatInteger(page.total)} super-vilain{page.total > 1 ? "s" : ""}
          </p>
          <div className="overflow-x-auto border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)]">
            <table className="w-full text-left text-sm">
              <thead className="border-b-2 border-border text-muted">
                <tr>
                  <th scope="col" className="px-4 py-3 font-medium">
                    Alias
                  </th>
                  <th scope="col" className="px-4 py-3 font-medium">
                    Menace
                  </th>
                  <th scope="col" className="px-4 py-3 font-medium">
                    Statut
                  </th>
                  <th scope="col" className="px-4 py-3 text-right font-medium">
                    Incidents
                  </th>
                </tr>
              </thead>
              <tbody>
                {page.items.map((villain) => (
                  <tr key={villain.id} className="border-b border-border last:border-0">
                    <th scope="row" className="px-4 py-3 font-semibold">
                      <Link
                        href={adminVillainRoute(villain.id)}
                        className="text-accent hover:underline"
                      >
                        {villain.alias}
                      </Link>
                    </th>
                    <td className="px-4 py-3">
                      <Badge tone={threatTone[villain.threatLevel]}>
                        {threatLevelLabels[villain.threatLevel]}
                      </Badge>
                    </td>
                    <td className="px-4 py-3">{villainStatusLabels[villain.status]}</td>
                    <td className="px-4 py-3 text-right font-mono">
                      {formatInteger(villain.reportedIncidentsCount)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
          <Pagination page={page} basePath={routes.adminVillains} query={{ q: search }} />
        </>
      )}
    </>
  );
}
