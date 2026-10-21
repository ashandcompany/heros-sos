import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { moderateIncidentAction } from "@/app/actions/admin";
import { ArbitrationForm } from "@/components/admin/ArbitrationForm";
import { decisionsFor } from "@/components/admin/decisions";
import { IncidentEditForm } from "@/components/admin/IncidentEditForm";
import { incidentDecisionLabels } from "@/components/admin/IncidentsModeration";
import { ModerationBadge } from "@/components/admin/ModerationBadge";
import { ModerationForm } from "@/components/admin/ModerationForm";
import { TeamMembersList } from "@/components/hero/TeamMembersList";
import { cn } from "@/components/ui/cn";
import {
  getIncident,
  getIncidentHeroes,
  getIncidentTypes,
  getVillains,
  incidentFeatures,
} from "@/lib/api";
import { formatDateTime, incidentStatusLabels, reporterRoleLabels } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { Incident } from "@/types";

export const metadata: Metadata = { title: "Incident" };

const panelClasses =
  "border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6";

/** Lit la session et les paramètres d'URL : rendu sous <Suspense>. */
async function IncidentEditor({ params }: { params: Promise<{ id: string }> }) {
  const token = await requireAdminSession();
  const { id } = await params;
  const [incident, villains, heroes, types] = await Promise.all([
    getIncident(token, id),
    getVillains(),
    getIncidentHeroes(),
    getIncidentTypes(token),
  ]);
  if (!incident) notFound();

  return (
    <>
      <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-display text-3xl font-bold">{incident.title}</h1>
          <p className="mt-2 text-sm text-muted">
            Signalé le{" "}
            <time dateTime={incident.reportedAt} className="font-mono">
              {formatDateTime(incident.reportedAt)}
            </time>
            {incident.reporterRole &&
              ` par un ${reporterRoleLabels[incident.reporterRole].toLowerCase()}`}
          </p>
        </div>
        {incidentFeatures.moderation && <ModerationBadge status={incident.moderation} />}
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
        <section aria-labelledby="edit-title" className={cn(panelClasses, "sm:p-8")}>
          <h2 id="edit-title" className="mb-6 font-display text-xl font-semibold">
            Informations de l&apos;incident
          </h2>
          {incidentFeatures.moderation ? (
            <IncidentEditForm
              incident={incident}
              villains={villains.map(({ id: villainId, alias }) => ({ id: villainId, alias }))}
              heroes={heroes.map(({ id: heroId, alias }) => ({ id: heroId, alias }))}
              types={types}
            />
          ) : (
            <IncidentFacts incident={incident} />
          )}
        </section>

        <div className="space-y-6">
          {incidentFeatures.moderation && (
            <section aria-labelledby="decision-title" className={panelClasses}>
              <h2 id="decision-title" className="font-display text-xl font-semibold">
                Décision
              </h2>
              <p className="mt-1 mb-4 text-sm text-muted">
                La déclaration est-elle conforme et réelle&nbsp;? Un incident rejeté n&apos;est plus
                diffusé.
              </p>
              <ModerationForm
                action={moderateIncidentAction}
                id={incident.id}
                subject={incident.title}
                decisions={decisionsFor(incident.moderation, incidentDecisionLabels)}
              />
            </section>
          )}

          {incident.team && (
            <section aria-labelledby="team-title" className={panelClasses}>
              <h2 id="team-title" className="mb-4 font-display text-xl font-semibold">
                Intervention en cours
              </h2>
              <TeamMembersList team={incident.team} />
              <h3 className="mt-6 mb-1 font-display font-semibold">Arbitrage</h3>
              <p className="mb-4 text-sm text-muted">
                Si un membre de l&apos;équipe a contesté le résultat proposé (vous en êtes prévenu
                par mail), tranchez : l&apos;intervention sera close.
              </p>
              <ArbitrationForm
                interventionRef={{ incidentId: incident.id, interventionId: incident.team.id }}
              />
            </section>
          )}
        </div>
      </div>
    </>
  );
}

/** Lecture seule, quand la correction n'est pas disponible (API). */
function IncidentFacts({ incident }: { incident: Incident }) {
  const place = [incident.location.address, incident.location.city].filter(Boolean).join(", ");
  return (
    <>
      {incident.description ? (
        <p className="text-pretty">{incident.description}</p>
      ) : (
        <p className="text-sm text-muted">Aucune description fournie.</p>
      )}
      <dl className="mt-6 grid gap-4 sm:grid-cols-2">
        <div>
          <dt className="text-sm text-muted">Type</dt>
          <dd>{incident.type.label}</dd>
        </div>
        <div>
          <dt className="text-sm text-muted">Statut</dt>
          <dd>{incidentStatusLabels[incident.status]}</dd>
        </div>
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
              <time dateTime={incident.occurredAt}>{formatDateTime(incident.occurredAt)}</time>
            </dd>
          </div>
        )}
      </dl>
    </>
  );
}

export default function AdminIncidentPage({ params }: PageProps<"/admin/incidents/[id]">) {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.adminIncidents} className="text-accent hover:underline">
          ← Incidents signalés
        </Link>
      </p>
      <Suspense
        fallback={
          <div
            aria-hidden="true"
            className="h-[40rem] border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        }
      >
        <IncidentEditor params={params} />
      </Suspense>
    </>
  );
}
