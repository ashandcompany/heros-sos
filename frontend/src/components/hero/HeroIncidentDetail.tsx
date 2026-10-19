import { notFound } from "next/navigation";
import { Badge } from "@/components/ui/Badge";
import { getHeroIncident } from "@/lib/api";
import {
  formatDateTime,
  incidentTypeLabels,
  interventionOutcomeLabels,
  reporterRoleLabels,
} from "@/lib/format";
import { requireHeroSession } from "@/lib/server-session";
import { FinishInterventionForm, PriorityForm, TakeChargeForm } from "./IncidentActions";
import { PriorityBadge, StatusBadge } from "./IncidentBadges";

const panelClasses =
  "border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6";

/** Lit la session et les paramètres d'URL : à rendre sous <Suspense>. */
export async function HeroIncidentDetail({ params }: { params: Promise<{ id: string }> }) {
  const { token, user } = await requireHeroSession();
  const { id } = await params;
  const incident = await getHeroIncident(token, id);
  if (!incident) notFound();

  const place = [incident.location.address, incident.location.district, incident.location.city]
    .filter(Boolean)
    .join(", ");
  const isMine = incident.status === "en-cours" && incident.hero?.id === user.id;
  const pastInterventions = incident.interventions.filter(({ outcome }) => outcome !== null);

  return (
    <>
      <div className="mb-8">
        <div className="flex flex-wrap items-center gap-2">
          <PriorityBadge priority={incident.priority} />
          <StatusBadge status={incident.status} />
          <Badge>{incidentTypeLabels[incident.type]}</Badge>
        </div>
        <h1 className="mt-3 font-display text-3xl font-bold">{incident.title}</h1>
        <p className="mt-2 text-sm text-muted">
          Signalé le{" "}
          <time dateTime={incident.reportedAt} className="font-mono">
            {formatDateTime(incident.reportedAt)}
          </time>
          {incident.reporterRole &&
            ` par un ${reporterRoleLabels[incident.reporterRole].toLowerCase()}`}
        </p>
      </div>

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
                <dd>{incident.villain?.alias ?? "Non identifié"}</dd>
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
              {incident.resolvedAt && (
                <div>
                  <dt className="text-sm text-muted">Résolu le</dt>
                  <dd>
                    <time dateTime={incident.resolvedAt}>
                      {formatDateTime(incident.resolvedAt)}
                    </time>
                  </dd>
                </div>
              )}
            </dl>
          </section>

          {pastInterventions.length > 0 && (
            <section aria-labelledby="history-title" className={panelClasses}>
              <h2 id="history-title" className="mb-4 font-display text-xl font-semibold">
                Interventions précédentes
              </h2>
              <ol className="space-y-3">
                {pastInterventions.map((intervention) => (
                  <li
                    key={intervention.id}
                    className="flex flex-wrap items-center justify-between gap-2 text-sm"
                  >
                    <span>
                      <span className="font-semibold">
                        {intervention.hero.id === user.id ? "Vous" : intervention.hero.alias}
                      </span>{" "}
                      ·{" "}
                      <time dateTime={intervention.startedAt} className="font-mono text-muted">
                        {formatDateTime(intervention.startedAt)}
                      </time>
                    </span>
                    {intervention.outcome && (
                      <Badge tone={intervention.outcome === "reussi" ? "success" : "danger"}>
                        {interventionOutcomeLabels[intervention.outcome]}
                      </Badge>
                    )}
                  </li>
                ))}
              </ol>
            </section>
          )}
        </div>

        <div className="space-y-6">
          <section aria-labelledby="intervention-title" className={panelClasses}>
            <h2 id="intervention-title" className="font-display text-xl font-semibold">
              Intervention
            </h2>
            <div className="mt-4">
              {incident.status === "signale" && (
                <>
                  <p className="mb-4 text-sm text-muted">
                    Aucun super-héros n&apos;intervient. Un seul héros peut prendre en charge
                    l&apos;incident à la fois.
                  </p>
                  <TakeChargeForm incidentId={incident.id} />
                </>
              )}
              {isMine && (
                <>
                  <p className="mb-4 text-sm">
                    <span className="font-semibold text-accent">Vous intervenez</span> sur cet
                    incident. Indiquez le résultat une fois l&apos;intervention terminée.
                  </p>
                  <FinishInterventionForm incidentId={incident.id} />
                </>
              )}
              {incident.status === "en-cours" && !isMine && (
                <p className="text-sm">
                  Déjà pris en charge par{" "}
                  <span className="font-semibold">{incident.hero?.alias ?? "un super-héros"}</span>.
                </p>
              )}
              {incident.status === "resolu" && (
                <p className="text-sm">
                  Incident résolu
                  {incident.hero && (
                    <>
                      {" "}
                      par{" "}
                      <span className="font-semibold">
                        {incident.hero.id === user.id ? "vous" : incident.hero.alias}
                      </span>
                    </>
                  )}
                  .
                </p>
              )}
            </div>
          </section>

          {incident.status !== "resolu" && (
            <section aria-labelledby="priority-title" className={panelClasses}>
              <h2 id="priority-title" className="mb-4 font-display text-xl font-semibold">
                Priorité
              </h2>
              <PriorityForm incidentId={incident.id} priority={incident.priority} />
            </section>
          )}
        </div>
      </div>
    </>
  );
}
