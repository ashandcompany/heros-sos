import Link from "next/link";
import { formatDateTime, incidentTypeLabels } from "@/lib/format";
import { heroIncidentRoute } from "@/lib/routes";
import type { Incident } from "@/types";
import { PriorityBadge, StatusBadge } from "./IncidentBadges";

/**
 * Signalement vu par un super-héros : priorité, statut et héros qui intervient.
 * Le titre est un lien vers le détail ; la carte entière est cliquable.
 */
export function HeroIncidentCard({ incident, heroId }: { incident: Incident; heroId: string }) {
  const place = [incident.location.address, incident.location.city].filter(Boolean).join(", ");
  const takenBy =
    incident.status === "en-cours" && incident.hero
      ? incident.hero.id === heroId
        ? "Vous intervenez"
        : `Pris en charge par ${incident.hero.alias}`
      : null;

  return (
    <article className="relative border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-4 focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-primary hover:border-muted sm:p-5">
      <div className="flex flex-wrap items-center gap-2">
        <PriorityBadge priority={incident.priority ?? "normale"} />
        <StatusBadge status={incident.status} />
        <span className="text-xs text-muted">{incidentTypeLabels[incident.type]}</span>
      </div>
      <h3 className="mt-3 font-display font-semibold">
        <Link
          href={heroIncidentRoute(incident.id)}
          className="after:absolute after:inset-0 after:rounded-2xl focus-visible:outline-none"
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
      <dl className="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-sm">
        <div className="flex gap-1">
          <dt className="text-muted">Vilain :</dt>
          <dd className="font-medium">{incident.villain?.alias ?? "Non identifié"}</dd>
        </div>
        {takenBy && (
          <div className="flex gap-1">
            <dt className="sr-only">Intervention :</dt>
            <dd
              className={incident.hero?.id === heroId ? "font-semibold text-accent" : "font-medium"}
            >
              {takenBy}
            </dd>
          </div>
        )}
      </dl>
    </article>
  );
}
