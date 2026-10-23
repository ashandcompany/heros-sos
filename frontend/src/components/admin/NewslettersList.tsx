import Link from "next/link";
import { Badge } from "@/components/ui/Badge";
import { getNewsletterIssues } from "@/lib/api";
import { formatDateTime, formatInteger } from "@/lib/format";
import { adminNewsletterRoute, routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import { EmptyState } from "./ModerationFilter";
import { Pagination, readPageNumber, type PageSearchParams } from "./Pagination";

const PAGE_SIZE = 20;

/** Numéros de la newsletter, brouillons et envoyés. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function NewslettersList({ searchParams }: { searchParams: PageSearchParams }) {
  const token = await requireAdminSession();
  const pageNumber = await readPageNumber(searchParams);
  const page = await getNewsletterIssues(token, {
    limit: PAGE_SIZE,
    offset: (pageNumber - 1) * PAGE_SIZE,
  });

  if (page.total === 0) {
    return <EmptyState>Aucune newsletter pour le moment. Rédigez la première&nbsp;!</EmptyState>;
  }

  return (
    <>
      <p className="mb-3 text-sm text-muted">
        {formatInteger(page.total)} newsletter{page.total > 1 ? "s" : ""}
      </p>
      <ul className="space-y-3">
        {page.items.map((issue) => {
          const titleId = `newsletter-${issue.id}-titre`;
          return (
            <li key={issue.id}>
              <article
                aria-labelledby={titleId}
                className="grid gap-4 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
              >
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge tone={issue.sentAt ? "success" : "paper"}>
                      {issue.sentAt ? "Envoyée" : "Brouillon"}
                    </Badge>
                    {issue.recipientUserId && <Badge tone="warning">Un seul destinataire</Badge>}
                  </div>
                  <h2 id={titleId} className="mt-3 font-display text-lg font-semibold">
                    {issue.subject}
                  </h2>
                  <p className="mt-2 text-xs text-muted">
                    {issue.sentAt ? (
                      <>
                        Envoyée le{" "}
                        <time dateTime={issue.sentAt} className="font-mono">
                          {formatDateTime(issue.sentAt)}
                        </time>
                      </>
                    ) : (
                      <>
                        Créée le{" "}
                        <time dateTime={issue.createdAt} className="font-mono">
                          {formatDateTime(issue.createdAt)}
                        </time>
                      </>
                    )}
                  </p>
                </div>
                <Link
                  href={adminNewsletterRoute(issue.id)}
                  className="inline-flex min-h-11 items-center justify-center rounded-md border border-border px-4 text-sm font-semibold hover:border-muted hover:bg-surface-raised"
                >
                  {issue.sentAt ? "Consulter" : "Modifier"}
                  <span className="sr-only"> : {issue.subject}</span>
                </Link>
              </article>
            </li>
          );
        })}
      </ul>
      <Pagination page={page} basePath={routes.adminNewsletters} />
    </>
  );
}
