import Link from "next/link";
import { moderateIncidentAction } from "@/app/actions/admin";
import { Badge } from "@/components/ui/Badge";
import { getAllIncidents } from "@/lib/api";
import { formatDateTime, incidentStatusLabels, reporterRoleLabels } from "@/lib/format";
import { adminIncidentRoute, routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { Incident } from "@/types";
import { decisionsFor } from "./decisions";
import { ModerationBadge } from "./ModerationBadge";
import {
  EmptyState,
  ModerationFilter,
  readModerationFilter,
  type ModerationSearchParams,
} from "./ModerationFilter";
import { ModerationForm } from "./ModerationForm";

export const incidentDecisionLabels = {
  approve: "Valider",
  reject: "Rejeter",
  restore: "Rétablir",
};

function IncidentModerationCard({ incident }: { incident: Incident }) {
  const place = [incident.location.address, incident.location.city].filter(Boolean).join(", ");
  const titleId = `incident-${incident.id}-titre`;

  return (
    <article
      aria-labelledby={titleId}
      className="grid gap-5 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto]"
    >
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <ModerationBadge status={incident.moderation} />
          <Badge>{incidentStatusLabels[incident.status]}</Badge>
          <Badge>{incident.type.label}</Badge>
        </div>
        <h2 id={titleId} className="mt-3 font-display text-lg font-semibold">
          {incident.title}
        </h2>
        <p className="mt-1 text-sm text-muted">
          {place} · Signalé le{" "}
          <time dateTime={incident.reportedAt} className="font-mono">
            {formatDateTime(incident.reportedAt)}
          </time>
          {incident.reporterRole &&
            ` · Déclarant : ${reporterRoleLabels[incident.reporterRole].toLowerCase()}`}
        </p>
        {incident.description ? (
          <p className="mt-3 text-sm text-pretty">{incident.description}</p>
        ) : (
          <p className="mt-3 text-sm text-muted italic">Aucune description.</p>
        )}
        <dl className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm">
          <div className="flex gap-1">
            <dt className="text-muted">Vilain :</dt>
            <dd className="font-medium">{incident.villain?.alias ?? "Non identifié"}</dd>
          </div>
          <div className="flex gap-1">
            <dt className="text-muted">Super-héros :</dt>
            <dd className="font-medium">{incident.hero?.alias ?? "En attente"}</dd>
          </div>
        </dl>
      </div>

      <div className="flex flex-col gap-3 lg:items-end">
        <ModerationForm
          action={moderateIncidentAction}
          id={incident.id}
          subject={incident.title}
          decisions={decisionsFor(incident.moderation, incidentDecisionLabels)}
          className="lg:[&>div]:justify-end"
        />
        <Link
          href={adminIncidentRoute(incident.id)}
          className="inline-flex min-h-11 items-center text-sm font-semibold text-accent hover:underline"
        >
          Corriger les informations<span className="sr-only"> : {incident.title}</span>
        </Link>
      </div>
    </article>
  );
}

/** File de modération des incidents. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function IncidentsModeration({
  searchParams,
}: {
  searchParams: ModerationSearchParams;
}) {
  const token = await requireAdminSession();
  const filter = await readModerationFilter(searchParams);
  const incidents = await getAllIncidents(token, filter);

  return (
    <>
      <ModerationFilter
        basePath={routes.adminIncidents}
        current={filter}
        labels={{ "en-attente": "À vérifier", valide: "Validés", rejete: "Rejetés", tous: "Tous" }}
      />
      {incidents.length === 0 ? (
        <EmptyState>
          {filter === "en-attente"
            ? "Aucun incident en attente de vérification."
            : "Aucun incident dans cette catégorie."}
        </EmptyState>
      ) : (
        <ul className="space-y-4">
          {incidents.map((incident) => (
            <li key={incident.id}>
              <IncidentModerationCard incident={incident} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
