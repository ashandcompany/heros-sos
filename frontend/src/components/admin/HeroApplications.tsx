import { reviewHeroApplicationAction } from "@/app/actions/admin";
import { Avatar } from "@/components/ui/Avatar";
import { countries } from "@/lib/countries";
import { getHeroApplications } from "@/lib/api";
import { formatDate, formatDateTime, formatFullName, formatPhone, phoneHref } from "@/lib/format";
import { requireAdminSession } from "@/lib/server-session";
import { civilityOptions } from "@/lib/validation/signup";
import type { HeroApplication } from "@/types";
import { EmptyState } from "./ModerationFilter";
import { ModerationForm } from "./ModerationForm";

function ApplicationCard({ application }: { application: HeroApplication }) {
  const titleId = `candidature-${application.id}-titre`;
  const civility = civilityOptions.find(({ value }) => value === application.civility)?.label;
  const country = countries.find(({ code }) => code === application.address.country)?.name;

  return (
    <article
      aria-labelledby={titleId}
      className="border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 sm:p-6"
    >
      <div className="flex items-start gap-4">
        <Avatar name={application.alias} src={null} size={56} />
        <div className="min-w-0 flex-1">
          <h3 id={titleId} className="font-display text-xl font-bold">
            {application.alias}
          </h3>
          <p className="text-sm text-muted">{application.power}</p>
          <p className="mt-1 text-xs text-muted">
            Demande déposée le{" "}
            <time dateTime={application.submittedAt} className="font-mono">
              {formatDateTime(application.submittedAt)}
            </time>
          </p>
        </div>
      </div>

      <blockquote className="mt-4 border-l-2 border-accent pl-4 text-sm text-pretty">
        {application.motivation}
      </blockquote>

      <dl className="mt-5 grid gap-x-6 gap-y-3 border-t border-border pt-5 text-sm sm:grid-cols-2">
        <div>
          <dt className="text-muted">Identité civile</dt>
          <dd>
            {civility && civility !== "Ne pas préciser" ? `${civility} ` : ""}
            {formatFullName(application.lastName, application.firstName)}
          </dd>
        </div>
        <div>
          <dt className="text-muted">Date de naissance</dt>
          <dd>
            <time dateTime={application.birthDate}>{formatDate(application.birthDate)}</time>
          </dd>
        </div>
        <div>
          <dt className="text-muted">Contact</dt>
          <dd className="break-all">
            <a href={`mailto:${application.email}`} className="text-accent hover:underline">
              {application.email}
            </a>
            <br />
            <a href={phoneHref(application.mobilePhone)} className="text-accent hover:underline">
              {formatPhone(application.mobilePhone)}
            </a>
          </dd>
        </div>
        <div>
          <dt className="text-muted">Adresse</dt>
          <dd>
            <address className="not-italic">
              {application.address.street}
              <br />
              {application.address.postalCode} {application.address.city}
              {country && ` (${country})`}
            </address>
          </dd>
        </div>
      </dl>

      <ModerationForm
        action={reviewHeroApplicationAction}
        id={application.id}
        subject={`candidature de ${application.alias}`}
        className="mt-5"
        decisions={[
          { value: "valide", label: "Valider la demande", variant: "primary" },
          { value: "rejete", label: "Refuser", variant: "danger" },
        ]}
      />
    </article>
  );
}

/** Demandes d'inscription en attente. Lit la session : à rendre sous <Suspense>. */
export async function HeroApplications() {
  const token = await requireAdminSession();
  const applications = await getHeroApplications(token, "en-attente");

  if (applications.length === 0) {
    return <EmptyState>Aucune demande de super-héros en attente.</EmptyState>;
  }

  return (
    <ul className="grid gap-4 lg:grid-cols-2">
      {applications.map((application) => (
        <li key={application.id}>
          <ApplicationCard application={application} />
        </li>
      ))}
    </ul>
  );
}
