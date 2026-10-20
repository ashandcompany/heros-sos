import Link from "next/link";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { IncidentMap } from "@/components/map/IncidentMap";
import { MapLegend } from "@/components/map/MapLegend";
import { getIncidentMap } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { getMapTileConfig } from "@/lib/map";
import { heroIncidentRoute } from "@/lib/routes";
import { requireHeroSession } from "@/lib/server-session";
import { PriorityBadge, StatusBadge } from "./IncidentBadges";

/** Carte opérationnelle et sa liste. Lit la session : à rendre sous <Suspense>. */
export async function HeroIncidentMapContent() {
  const { token, user } = await requireHeroSession();
  const data = await getIncidentMap(token);
  const tiles = getMapTileConfig();
  const heroByIncident = new Map(data.teams.map((team) => [team.incidentId, team.hero]));

  return (
    <div className="space-y-6">
      <MapLegend showTeams />
      <IncidentMap data={data} tiles={tiles} />

      <p role="status" className="text-sm text-muted">
        {data.incidents.length} incident{data.incidents.length > 1 ? "s" : ""} non résolu
        {data.incidents.length > 1 ? "s" : ""} sur la carte, {data.teams.length} super-héros en
        intervention.
        {data.unlocatedCount > 0 &&
          ` ${data.unlocatedCount} incident${data.unlocatedCount > 1 ? "s" : ""} sans position connue n'${data.unlocatedCount > 1 ? "apparaissent" : "apparaît"} pas sur la carte.`}{" "}
        La position d&apos;un super-héros est celle de l&apos;incident qu&apos;il traite.
      </p>

      <section aria-labelledby="map-list-title">
        <h2 id="map-list-title" className="mb-3 font-display text-lg font-semibold">
          Incidents de la carte
        </h2>
        {data.incidents.length === 0 ? (
          <EmptyState>Aucun incident non résolu pour le moment.</EmptyState>
        ) : (
          <ul className="divide-y divide-border border-2 border-border bg-surface">
            {data.incidents.map((incident) => {
              const hero = heroByIncident.get(incident.id);
              return (
                <li key={incident.id} className="flex flex-wrap items-center gap-x-4 gap-y-1 p-3">
                  <span className="flex gap-2">
                    <PriorityBadge priority={incident.priority} />
                    <StatusBadge status={incident.status} />
                  </span>
                  <Link
                    href={heroIncidentRoute(incident.id)}
                    className="font-medium underline-offset-2 hover:underline"
                  >
                    {incident.title}
                  </Link>
                  <span className="text-sm text-muted">
                    {incident.type.label} · {incident.place} ·{" "}
                    <time dateTime={incident.reportedAt} className="font-mono">
                      {formatDateTime(incident.reportedAt)}
                    </time>
                  </span>
                  {hero && (
                    <span
                      className={
                        hero.id === user.id ? "text-sm font-semibold text-accent" : "text-sm"
                      }
                    >
                      {hero.id === user.id ? "Vous intervenez" : `${hero.alias} intervient`}
                    </span>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </div>
  );
}

export function HeroIncidentMapSkeleton() {
  return (
    <div aria-hidden="true" className="space-y-6">
      <div className="h-6 w-2/3 bg-surface motion-safe:animate-pulse" />
      <div className="h-[min(70vh,560px)] border-2 border-border bg-surface motion-safe:animate-pulse" />
    </div>
  );
}
