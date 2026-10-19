import { Badge } from "@/components/ui/Badge";
import {
  formatDateTime,
  incidentStatusLabels,
  incidentTypeLabels,
  reporterRoleLabels,
} from "@/lib/format";
import type { Incident, IncidentStatus } from "@/types";

const statusTone: Record<IncidentStatus, "warning" | "accent" | "success"> = {
  signale: "warning",
  "en-cours": "accent",
  resolu: "success",
};

interface IncidentListItemProps {
  incident: Incident;
  /** Affiche aussi la description et le rôle du déclarant (liste « Mes déclarations »). */
  detailed?: boolean;
}

export function IncidentListItem({ incident, detailed = false }: IncidentListItemProps) {
  const place = [incident.location.address, incident.location.city].filter(Boolean).join(", ");

  return (
    <article className="rounded-2xl border border-border bg-surface p-4 sm:p-5">
      <div className="flex flex-wrap items-center gap-2">
        <Badge tone={statusTone[incident.status]}>{incidentStatusLabels[incident.status]}</Badge>
        <Badge>{incidentTypeLabels[incident.type]}</Badge>
        {detailed && incident.reporterRole && (
          <span className="text-xs text-muted">
            Vous étiez {reporterRoleLabels[incident.reporterRole].toLowerCase()}
          </span>
        )}
      </div>
      <h3 className="mt-3 font-display font-semibold">{incident.title}</h3>
      <p className="mt-1 text-sm text-muted">
        <span className="sr-only">Lieu : </span>
        {place}
      </p>
      <p className="text-sm text-muted">
        Signalé le <time dateTime={incident.reportedAt}>{formatDateTime(incident.reportedAt)}</time>
      </p>
      {detailed && incident.description && (
        <p className="mt-3 text-sm text-pretty">{incident.description}</p>
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
    </article>
  );
}
