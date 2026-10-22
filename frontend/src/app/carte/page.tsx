import type { Metadata } from "next";
import { Suspense } from "react";
import { IncidentMapExplorer } from "@/components/map/IncidentMapExplorer";
import { MapExplorerSkeleton } from "@/components/map/MapExplorerSkeleton";
import { Container } from "@/components/ui/Container";
import { getPublicIncidentMap } from "@/lib/api";
import { getMapTileConfig } from "@/lib/map";

export const metadata: Metadata = {
  title: "Carte des incidents",
  description:
    "Les incidents en cours signalés sur Heros-SOS et vérifiés par la modération, ville par ville.",
};

async function PublicIncidentMap() {
  const data = await getPublicIncidentMap();
  return <IncidentMapExplorer data={data} tiles={getMapTileConfig()} variant="public" />;
}

export default function IncidentMapPage() {
  return (
    <Container className="py-8 sm:py-10">
      <h1 className="font-display text-3xl font-bold sm:text-4xl">Carte des incidents</h1>
      <p className="mt-2 mb-6 max-w-2xl text-muted">
        Les incidents en cours, vérifiés par l&apos;équipe de modération. Choisissez un incident
        dans la liste ou un repère sur la carte pour en savoir plus.
      </p>
      <Suspense fallback={<MapExplorerSkeleton />}>
        <PublicIncidentMap />
      </Suspense>
    </Container>
  );
}
