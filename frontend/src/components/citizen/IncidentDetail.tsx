import Link from "next/link";
import { notFound } from "next/navigation";
import type { ReactNode } from "react";
import { Badge } from "@/components/ui/Badge";
import { RatingStars } from "@/components/ui/RatingStars";
import { getCitizenIncident } from "@/lib/api";
import {
  formatDateTime,
  incidentStatusLabels,
  incidentTypeLabels,
  reporterRoleLabels,
} from "@/lib/format";
import { heroSheetRoute, routes, villainSheetRoute } from "@/lib/routes";
import { requireCitizenSession } from "@/lib/server-session";
import type { CitizenIncidentDetail } from "@/types";
import { statusTone } from "./IncidentListItem";

const panelClasses =
  "border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6";

type StepState = "done" | "current" | "pending" | "failed";

interface TimelineStep {
  key: string;
  label: ReactNode;
  date?: string;
  state: StepState;
}

const markerClasses: Record<StepState, string> = {
  done: "border-success bg-success",
  current: "border-rating bg-rating motion-safe:animate-pulse",
  pending: "border-border bg-background",
  failed: "border-brick bg-brick",
};

const stateLabels: Record<StepState, string> = {
  done: "terminé",
  current: "en cours",
  pending: "à venir",
  failed: "échec",
};

/** Étapes du suivi, déduites du statut, de la modération et des interventions. */
function timelineSteps({ incident, interventions }: CitizenIncidentDetail): TimelineStep[] {
  const steps: TimelineStep[] = [
    { key: "signale", label: "Signalement reçu", date: incident.reportedAt, state: "done" },
  ];

  if (incident.moderation === "rejete") {
    steps.push({
      key: "moderation",
      label: "Signalement rejeté par la modération",
      state: "failed",
    });
    return steps;
  }
  steps.push(
    incident.moderation === "valide"
      ? { key: "moderation", label: "Signalement vérifié par la modération", state: "done" }
      : { key: "moderation", label: "Vérification par la modération", state: "current" },
  );

  for (const intervention of interventions) {
    steps.push({
      key: `${intervention.id}-start`,
      label: <>Pris en charge par {intervention.hero.alias}</>,
      date: intervention.startedAt,
      state: "done",
    });
    if (intervention.outcome === "echoue") {
      steps.push({
        key: `${intervention.id}-end`,
        label: (
          <>Intervention de {intervention.hero.alias} sans succès, incident de nouveau proposé</>
        ),
        date: intervention.endedAt ?? undefined,
        state: "failed",
      });
    }
  }

  // Héros affecté par la modération, sans prise en charge enregistrée.
  const hero = incident.hero;
  if (hero && !interventions.some((intervention) => intervention.hero.id === hero.id)) {
    steps.push({ key: "affectation", label: <>Pris en charge par {hero.alias}</>, state: "done" });
  }

  if (incident.status === "resolu") {
    steps.push({
      key: "resolu",
      label: "Incident résolu",
      date: incident.resolvedAt,
      state: "done",
    });
  } else if (incident.status === "en-cours") {
    steps.push({ key: "resolu", label: "Intervention en cours", state: "current" });
  } else {
    steps.push({ key: "resolu", label: "En attente d'un super-héros", state: "pending" });
  }

  return steps;
}

function Timeline({ steps }: { steps: TimelineStep[] }) {
  return (
    <ol className="space-y-0">
      {steps.map((step, index) => (
        <li key={step.key} className="relative flex gap-4 pb-5 last:pb-0">
          {index < steps.length - 1 && (
            <span
              aria-hidden="true"
              className="absolute top-4 left-[0.4375rem] h-full w-0.5 bg-border"
            />
          )}
          <span
            aria-hidden="true"
            className={`relative mt-1 size-4 shrink-0 rounded-full border-2 ${markerClasses[step.state]}`}
          />
          <div>
            <p className={step.state === "pending" ? "text-muted" : "font-medium"}>
              {step.label}
              <span className="sr-only"> ({stateLabels[step.state]})</span>
            </p>
            {step.date && (
              <time dateTime={step.date} className="font-mono text-sm text-muted">
                {formatDateTime(step.date)}
              </time>
            )}
          </div>
        </li>
      ))}
    </ol>
  );
}

/** Détail d'un incident. Lit la session et les paramètres d'URL : à rendre sous <Suspense>. */
export async function IncidentDetail({ params }: { params: Promise<{ id: string }> }) {
  const token = await requireCitizenSession();
  const { id } = await params;
  const detail = await getCitizenIncident(token, id);
  if (!detail) notFound();

  const { incident, isMine, rating } = detail;
  const place = [incident.location.address, incident.location.district, incident.location.city]
    .filter(Boolean)
    .join(", ");

  return (
    <>
      <header className="mb-8">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={statusTone[incident.status]}>{incidentStatusLabels[incident.status]}</Badge>
          <Badge>{incidentTypeLabels[incident.type]}</Badge>
          {isMine && incident.moderation === "en-attente" && (
            <Badge tone="warning">En cours de vérification</Badge>
          )}
          {isMine && incident.moderation === "rejete" && (
            <Badge tone="danger">Rejeté par la modération</Badge>
          )}
        </div>
        <h1 className="mt-3 font-display text-3xl font-bold text-balance">{incident.title}</h1>
        <p className="mt-2 text-sm text-muted">
          Signalé le{" "}
          <time dateTime={incident.reportedAt} className="font-mono">
            {formatDateTime(incident.reportedAt)}
          </time>
          {isMine && incident.reporterRole
            ? ` · Vous l'avez déclaré en tant que ${reporterRoleLabels[incident.reporterRole].toLowerCase()}`
            : " · Déclaré par un autre citoyen"}
        </p>
      </header>

      {isMine && incident.moderation === "rejete" && (
        <p role="note" className="mb-6 border-2 border-brick bg-brick/10 px-4 py-3 text-sm">
          La modération a jugé ce signalement non conforme : il n&apos;est pas diffusé aux
          super-héros. Si vous pensez qu&apos;il s&apos;agit d&apos;une erreur, vous pouvez le
          déclarer à nouveau en précisant le lieu et les faits.
        </p>
      )}

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
        <div className="space-y-6">
          <section aria-labelledby="report-title" className={panelClasses}>
            <h2 id="report-title" className="mb-4 font-display text-xl font-semibold">
              Signalement
            </h2>
            {incident.description ? (
              <p className="text-pretty">{incident.description}</p>
            ) : (
              <p className="text-sm text-muted">Aucune description fournie.</p>
            )}
            <dl className="mt-6 grid gap-4 sm:grid-cols-2">
              <div>
                <dt className="text-sm text-muted">Lieu</dt>
                <dd>{place}</dd>
              </div>
              <div>
                <dt className="text-sm text-muted">Super-vilain</dt>
                <dd>
                  {incident.villain ? (
                    <Link
                      href={villainSheetRoute(incident.villain.id)}
                      className="text-accent hover:underline"
                    >
                      {incident.villain.alias}
                    </Link>
                  ) : (
                    "Non identifié"
                  )}
                </dd>
              </div>
              {incident.occurredAt && (
                <div>
                  <dt className="text-sm text-muted">Date des faits</dt>
                  <dd>
                    <time dateTime={incident.occurredAt}>
                      {formatDateTime(incident.occurredAt)}
                    </time>
                  </dd>
                </div>
              )}
            </dl>
          </section>

          <section aria-labelledby="timeline-title" className={panelClasses}>
            <h2 id="timeline-title" className="mb-5 font-display text-xl font-semibold">
              Suivi
            </h2>
            <Timeline steps={timelineSteps(detail)} />
          </section>
        </div>

        <div className="space-y-6">
          <section aria-labelledby="hero-title" className={panelClasses}>
            <h2 id="hero-title" className="mb-3 font-display text-xl font-semibold">
              Super-héros
            </h2>
            {incident.hero ? (
              <p>
                <Link
                  href={heroSheetRoute(incident.hero.id)}
                  className="font-semibold text-accent hover:underline"
                >
                  {incident.hero.alias}
                </Link>{" "}
                {incident.status === "resolu"
                  ? "a résolu cet incident."
                  : "intervient en ce moment."}
              </p>
            ) : (
              <p className="text-sm text-muted">
                {incident.moderation === "rejete"
                  ? "Aucun super-héros n'interviendra sur ce signalement."
                  : "Aucun super-héros n'intervient encore. Les héros à proximité ont été prévenus."}
              </p>
            )}
          </section>

          {isMine && incident.status === "resolu" && incident.hero && (
            <section aria-labelledby="rating-title" className={panelClasses}>
              <h2 id="rating-title" className="mb-3 font-display text-xl font-semibold">
                Votre avis
              </h2>
              {rating ? (
                <>
                  <RatingStars rating={rating.rating} />
                  {rating.comment && (
                    <blockquote className="mt-3 border-l-2 border-border pl-3 text-sm text-pretty">
                      {rating.comment}
                    </blockquote>
                  )}
                  {rating.reply && (
                    <div className="mt-4 border-2 border-border bg-background p-3">
                      <p className="text-sm font-semibold">Réponse de {incident.hero.alias}</p>
                      <p className="mt-1 text-sm text-pretty">{rating.reply.content}</p>
                    </div>
                  )}
                </>
              ) : (
                <>
                  <p className="mb-4 text-sm text-muted">
                    Comment s&apos;est passée l&apos;intervention de {incident.hero.alias}&nbsp;?
                  </p>
                  <Link
                    href={`${routes.profileInterventions}#intervention-${incident.id}`}
                    className="font-semibold text-accent hover:underline"
                  >
                    Noter l&apos;intervention<span aria-hidden="true">&nbsp;→</span>
                  </Link>
                </>
              )}
            </section>
          )}
        </div>
      </div>
    </>
  );
}
