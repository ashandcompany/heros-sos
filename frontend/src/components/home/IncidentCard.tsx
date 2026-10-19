import { Badge } from "@/components/ui/Badge";
import { formatDate, formatLocation, incidentTypeLabels } from "@/lib/format";
import type { ResolvedIncident } from "@/types";

function PinIcon() {
  return (
    <svg viewBox="0 0 20 20" aria-hidden="true" className="size-4 shrink-0 fill-current">
      <path d="M10 1.5a6 6 0 0 0-6 6c0 4.5 6 11 6 11s6-6.5 6-11a6 6 0 0 0-6-6Zm0 8.25a2.25 2.25 0 1 1 0-4.5 2.25 2.25 0 0 1 0 4.5Z" />
    </svg>
  );
}

export function IncidentCard({ incident }: { incident: ResolvedIncident }) {
  return (
    <article className="flex h-full flex-col rounded-2xl border border-border bg-surface p-5 sm:p-6">
      <div className="flex flex-wrap items-center gap-2">
        <Badge tone="accent">{incidentTypeLabels[incident.type]}</Badge>
        <Badge tone="success">
          <span aria-hidden="true">✓</span> Résolu
        </Badge>
      </div>

      <h3 className="mt-4 font-display text-lg leading-snug font-semibold text-balance">
        {incident.title}
      </h3>

      <p className="mt-3 flex items-center gap-1.5 text-sm text-muted">
        <PinIcon />
        <span className="sr-only">Lieu :</span>
        {formatLocation(incident.location)}
      </p>
      <p className="mt-1 mb-6 text-sm text-muted">
        Traité le <time dateTime={incident.resolvedAt}>{formatDate(incident.resolvedAt)}</time>
      </p>

      <dl className="mt-auto grid grid-cols-2 gap-4 border-t border-border pt-4 text-sm">
        <div>
          <dt className="text-muted">Super-héros</dt>
          <dd className="mt-0.5 font-semibold text-foreground">{incident.hero.alias}</dd>
        </div>
        <div>
          <dt className="text-muted">Vilain</dt>
          <dd className="mt-0.5 font-semibold text-primary">{incident.villain.alias}</dd>
        </div>
      </dl>
    </article>
  );
}
