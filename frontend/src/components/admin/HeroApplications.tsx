import { Avatar } from "@/components/ui/Avatar";
import { getHeroApplications } from "@/lib/api";
import { formatDateTime, formatFullName, formatInteger } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { HeroApplication } from "@/types";
import { HeroReviewForm } from "./HeroReviewForm";
import { EmptyState } from "./ModerationFilter";
import { Pagination, readPageNumber, type PageSearchParams } from "./Pagination";

const PAGE_SIZE = 10;

function ApplicationCard({ application }: { application: HeroApplication }) {
  const titleId = `candidature-${application.userId}-titre`;

  return (
    <article
      aria-labelledby={titleId}
      className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6"
    >
      <div className="flex items-start gap-4">
        <Avatar name={application.alias} src={null} size={56} />
        <div className="min-w-0 flex-1">
          <h3 id={titleId} className="font-display text-xl font-bold">
            {application.alias}
          </h3>
          {application.bio && <p className="text-sm text-muted">{application.bio}</p>}
          <p className="mt-1 text-xs text-muted">
            Demande déposée le{" "}
            <time dateTime={application.submittedAt} className="font-mono">
              {formatDateTime(application.submittedAt)}
            </time>
          </p>
        </div>
      </div>

      {application.requestMessage && (
        <blockquote className="mt-4 border-l-2 border-accent pl-4 text-sm text-pretty">
          {application.requestMessage}
        </blockquote>
      )}

      <dl className="mt-5 grid gap-x-6 gap-y-3 border-t border-border pt-5 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-muted">Identité civile</dt>
          <dd>{formatFullName(application.lastName, application.firstName)}</dd>
        </div>
        <div>
          <dt className="text-muted">Contact</dt>
          <dd className="break-all">
            <a href={`mailto:${application.email}`} className="text-accent hover:underline">
              {application.email}
            </a>
          </dd>
        </div>
      </dl>

      <HeroReviewForm
        userId={application.userId}
        subject={`demande de ${application.alias}`}
        className="mt-5"
      />
    </article>
  );
}

/** Demandes en attente. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function HeroApplications({ searchParams }: { searchParams: PageSearchParams }) {
  const token = await requireAdminSession();
  const pageNumber = await readPageNumber(searchParams);
  const page = await getHeroApplications(token, {
    status: "en-attente",
    limit: PAGE_SIZE,
    offset: (pageNumber - 1) * PAGE_SIZE,
  });

  if (page.total === 0) {
    return <EmptyState>Aucune demande de super-héros en attente.</EmptyState>;
  }

  return (
    <>
      <p className="mb-4 text-sm text-muted">
        {formatInteger(page.total)} demande{page.total > 1 ? "s" : ""} en attente, des plus
        anciennes aux plus récentes.
      </p>
      <ul className="grid gap-4 lg:grid-cols-2">
        {page.items.map((application) => (
          <li key={application.userId}>
            <ApplicationCard application={application} />
          </li>
        ))}
      </ul>
      <Pagination page={page} basePath={routes.adminHeroes} />
    </>
  );
}
