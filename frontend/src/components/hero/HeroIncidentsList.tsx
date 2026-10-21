import Link from "next/link";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { cn } from "@/components/ui/cn";
import { getHeroIncidents } from "@/lib/api";
import { routes } from "@/lib/routes";
import { requireHeroSession } from "@/lib/server-session";
import type { IncidentStatus } from "@/types";
import { HeroIncidentCard } from "./HeroIncidentCard";

/** Filtres de la liste (`?statut=`) : « à traiter » par défaut, c'est-à-dire non résolus. */
const filters = {
  "a-traiter": { label: "À traiter", statuses: ["signale", "en-cours"] },
  signale: { label: "À prendre en charge", statuses: ["signale"] },
  "en-cours": { label: "Pris en charge", statuses: ["en-cours"] },
  resolu: { label: "Résolus", statuses: ["resolu"] },
  tous: { label: "Tous", statuses: undefined },
} satisfies Record<string, { label: string; statuses: IncidentStatus[] | undefined }>;

type Filter = keyof typeof filters;

const isFilter = (value: string | undefined): value is Filter => !!value && value in filters;

export type HeroIncidentsSearchParams = Promise<{ statut?: string | string[] }>;

/** Liste des signalements. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function HeroIncidentsList({
  searchParams,
}: {
  searchParams: HeroIncidentsSearchParams;
}) {
  const { token, user } = await requireHeroSession();
  const raw = (await searchParams).statut;
  const value = Array.isArray(raw) ? raw[0] : raw;
  const current: Filter = isFilter(value) ? value : "a-traiter";
  const incidents = await getHeroIncidents(token, filters[current].statuses);

  return (
    <>
      <nav aria-label="Filtrer par statut" className="mb-6">
        <ul className="flex flex-wrap gap-2">
          {(Object.keys(filters) as Filter[]).map((filter) => (
            <li key={filter}>
              <Link
                href={
                  filter === "a-traiter"
                    ? routes.heroIncidents
                    : `${routes.heroIncidents}?statut=${filter}`
                }
                aria-current={filter === current ? "page" : undefined}
                className={cn(
                  "inline-flex min-h-10 items-center rounded-full border px-4 text-sm font-medium",
                  filter === current
                    ? "border-primary bg-primary/15 text-foreground"
                    : "border-border text-muted hover:border-muted hover:text-foreground",
                )}
              >
                {filters[filter].label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      <p role="status" className="mb-4 text-sm text-muted">
        {incidents.length} incident{incidents.length > 1 ? "s" : ""}, triés par statut puis par
        priorité.
      </p>

      {incidents.length === 0 ? (
        <EmptyState>Aucun incident dans cette catégorie.</EmptyState>
      ) : (
        <ul className="grid gap-3 md:grid-cols-2">
          {incidents.map((incident) => (
            <li key={incident.id}>
              <HeroIncidentCard incident={incident} heroId={user.id} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
