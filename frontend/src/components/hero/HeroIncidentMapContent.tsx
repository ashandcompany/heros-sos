import { IncidentMapExplorer } from "@/components/map/IncidentMapExplorer";
import { MapExplorerSkeleton } from "@/components/map/MapExplorerSkeleton";
import { getIncidentMap } from "@/lib/api";
import { getMapTileConfig } from "@/lib/map";
import { requireHeroSession } from "@/lib/server-session";

/** Carte opérationnelle et sa liste. Lit la session : à rendre sous <Suspense>. */
export async function HeroIncidentMapContent() {
  const { token } = await requireHeroSession();
  const data = await getIncidentMap(token);
  return <IncidentMapExplorer data={data} tiles={getMapTileConfig()} variant="hero" />;
}

export const HeroIncidentMapSkeleton = MapExplorerSkeleton;
