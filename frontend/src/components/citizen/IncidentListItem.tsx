import Link from "next/link";
import { Badge } from "@/components/ui/Badge";
import { formatDateTime, incidentStatusLabels, reporterRoleLabels } from "@/lib/format";
import { citizenIncidentRoute } from "@/lib/routes";
import type { Incident, IncidentStatus } from "@/types";

export const statusTone: Record<IncidentStatus, "paper" | "warning" | "success"> = {
  signale: "paper",
  "en-cours": "warning",
  resolu: "success",
};

/** Héros inconnu : en attente, ou intervention dont le citoyen ne voit pas l'équipe (API). */
const heroPlaceholders: Record<IncidentStatus, string> = {
  signale: "En attente",
  "en-cours": "En intervention",
  resolu: "Intervention terminée",
};

interface IncidentListItemProps {
  incident: Incident;
  /** Affiche aussi la description et le rôle du déclarant (liste « Mes déclarations »). */
  detailed?: boolean;
}

export function IncidentListItem({ incident, detailed = false }: IncidentListItemProps) {
  const place = [incident.location.address, incident.location.city].filter(Boolean).join(", ");

  return (
    <article className="relative border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-4 focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-primary hover:border-muted sm:p-5">
      <div className="flex flex-wrap items-center gap-2">
        <Badge tone={statusTone[incident.status]}>{incidentStatusLabels[incident.status]}</Badge>
        <Badge>{incident.type.label}</Badge>
        {detailed && incident.moderation === "rejete" && (
          <Badge tone="danger">Rejeté par la modération</Badge>
        )}
        {detailed && incident.reporterRole && (
          <span className="text-xs text-muted">
            Vous étiez {reporterRoleLabels[incident.reporterRole].toLowerCase()}
          </span>
        )}
      </div>
      <h3 className="mt-3 font-display font-semibold">
        {/* Lien étiré : toute la carte mène au détail de l'incident. */}
        <Link
          href={citizenIncidentRoute(incident.id)}
          className="after:absolute after:inset-0 hover:underline focus-visible:outline-none"
        >
          {incident.title}
        </Link>
      </h3>
      <p className="mt-1 text-sm text-muted">
        <span className="sr-only">Lieu : </span>
        {place}
      </p>
      <p className="text-sm text-muted">
        Signalé le{" "}
        <time dateTime={incident.reportedAt} className="font-mono">
          {formatDateTime(incident.reportedAt)}
        </time>
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
          <dd className="font-medium">
            {incident.hero?.alias ?? heroPlaceholders[incident.status]}
          </dd>
        </div>
      </dl>
    </article>
  );
}
