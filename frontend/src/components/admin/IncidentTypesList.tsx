import { Badge } from "@/components/ui/Badge";
import { getAllIncidentTypes } from "@/lib/api";
import { formatInteger } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import { EmptyState } from "./ModerationFilter";
import { Pagination, readPageNumber, type PageSearchParams } from "./Pagination";

const PAGE_SIZE = 50;

/** Référentiel des types d'incident. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function IncidentTypesList({ searchParams }: { searchParams: PageSearchParams }) {
  const token = await requireAdminSession();
  const pageNumber = await readPageNumber(searchParams);
  const page = await getAllIncidentTypes(token, {
    limit: PAGE_SIZE,
    offset: (pageNumber - 1) * PAGE_SIZE,
  });

  if (page.total === 0) {
    return (
      <EmptyState>
        Aucun type d&apos;incident : tant que le référentiel est vide, personne ne peut déclarer
        d&apos;incident.
      </EmptyState>
    );
  }

  return (
    <>
      <p className="mb-3 text-sm text-muted">
        {formatInteger(page.total)} type{page.total > 1 ? "s" : ""}
      </p>
      <div className="overflow-x-auto border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)]">
        <table className="w-full text-left text-sm">
          <thead className="border-b-2 border-border text-muted">
            <tr>
              <th scope="col" className="px-4 py-3 font-medium">
                Libellé
              </th>
              <th scope="col" className="px-4 py-3 font-medium">
                Code
              </th>
              <th scope="col" className="px-4 py-3 font-medium">
                Danger
              </th>
            </tr>
          </thead>
          <tbody>
            {page.items.map((type) => (
              <tr key={type.id} className="border-b border-border last:border-0">
                <th scope="row" className="px-4 py-3 font-semibold">
                  {type.label}
                </th>
                <td className="px-4 py-3 font-mono">{type.code}</td>
                <td className="px-4 py-3">
                  {type.endangersLife ? (
                    <Badge tone="danger">Vies en danger</Badge>
                  ) : (
                    <span className="text-muted">—</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Pagination page={page} basePath={routes.adminIncidentTypes} />
    </>
  );
}
