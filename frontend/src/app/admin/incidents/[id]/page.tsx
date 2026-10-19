import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { moderateIncidentAction } from "@/app/actions/admin";
import { decisionsFor } from "@/components/admin/decisions";
import { IncidentEditForm } from "@/components/admin/IncidentEditForm";
import { incidentDecisionLabels } from "@/components/admin/IncidentsModeration";
import { ModerationBadge } from "@/components/admin/ModerationBadge";
import { ModerationForm } from "@/components/admin/ModerationForm";
import { getIncident, getSuperHeroes, getVillains } from "@/lib/api";
import { formatDateTime, reporterRoleLabels } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Corriger un incident" };

/** Lit la session et les paramètres d'URL : rendu sous <Suspense>. */
async function IncidentEditor({ params }: { params: Promise<{ id: string }> }) {
  const token = await requireAdminSession();
  const { id } = await params;
  const [incident, villains, heroes] = await Promise.all([
    getIncident(token, id),
    getVillains(),
    getSuperHeroes(),
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
        <ModerationBadge status={incident.moderation} />
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_18rem] lg:items-start">
        <section
          aria-labelledby="edit-title"
          className="border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 sm:p-8"
        >
          <h2 id="edit-title" className="mb-6 font-display text-xl font-semibold">
            Informations de l&apos;incident
          </h2>
          <IncidentEditForm
            incident={incident}
            villains={villains.map(({ id: villainId, alias }) => ({ id: villainId, alias }))}
            heroes={heroes.map(({ id: heroId, alias }) => ({ id: heroId, alias }))}
          />
        </section>

        <section
          aria-labelledby="decision-title"
          className="border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 sm:p-6"
        >
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
      </div>
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
