import type { Metadata } from "next";
import { Suspense } from "react";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { PriorityBadge, StatusBadge } from "@/components/hero/IncidentBadges";
import { IncidentMap } from "@/components/map/IncidentMap";
import { MapLegend } from "@/components/map/MapLegend";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { getPublicIncidentMap } from "@/lib/api";
import { formatDateTime, incidentTypeLabels } from "@/lib/format";
import { getMapTileConfig } from "@/lib/map";

export const metadata: Metadata = {
  title: "Carte des incidents",
  description:
    "Les incidents en cours signalés sur Heros-SOS et vérifiés par la modération, ville par ville.",
};

async function PublicIncidentMap() {
  const data = await getPublicIncidentMap();
  const tiles = getMapTileConfig();

  return (
    <div className="space-y-6">
      <MapLegend />
      <IncidentMap data={data} tiles={tiles} variant="public" />

      <p role="status" className="text-sm text-muted">
        {data.incidents.length} incident{data.incidents.length > 1 ? "s" : ""} en cours sur la
        carte. Les positions sont approximatives (au quartier près) pour protéger les personnes
        concernées.
      </p>

      <section aria-labelledby="public-map-list-title">
        <SectionHeading id="public-map-list-title" title="Incidents de la carte" className="mb-4" />
        {data.incidents.length === 0 ? (
          <EmptyState>Aucun incident en cours pour le moment.</EmptyState>
        ) : (
          <ul className="divide-y divide-border border-2 border-border bg-surface">
            {data.incidents.map((incident) => (
              <li key={incident.id} className="flex flex-wrap items-center gap-x-4 gap-y-1 p-3">
                <span className="flex gap-2">
                  <PriorityBadge priority={incident.priority} />
                  <StatusBadge status={incident.status} />
                </span>
                <span className="font-medium">{incident.title}</span>
                <span className="text-sm text-muted">
                  {incidentTypeLabels[incident.type]} · {incident.place} ·{" "}
                  <time dateTime={incident.reportedAt} className="font-mono">
                    {formatDateTime(incident.reportedAt)}
                  </time>
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

function MapSkeleton() {
  return (
    <div aria-hidden="true" className="space-y-6">
      <div className="h-6 w-2/3 bg-surface motion-safe:animate-pulse" />
      <div className="h-[min(70vh,560px)] border-2 border-border bg-surface motion-safe:animate-pulse" />
    </div>
  );
}

export default function IncidentMapPage() {
  return (
    <Container className="py-10 sm:py-14">
      <h1 className="font-display text-3xl font-bold sm:text-4xl">Carte des incidents</h1>
      <p className="mt-2 mb-8 max-w-2xl text-muted">
        Les incidents en cours, vérifiés par l&apos;équipe de modération, et les zones les plus
        touchées. Cliquez sur un repère pour en savoir plus.
      </p>
      <Suspense fallback={<MapSkeleton />}>
        <PublicIncidentMap />
      </Suspense>
    </Container>
  );
}
