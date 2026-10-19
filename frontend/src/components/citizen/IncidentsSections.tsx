import Link from "next/link";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { getMyIncidents, getOngoingIncidents, getVillains } from "@/lib/api";
import { routes } from "@/lib/routes";
import { requireCitizenSession } from "@/lib/server-session";
import { declareIncidentAction } from "@/app/actions/incidents";
import { DeclareIncidentForm } from "./DeclareIncidentForm";
import { IncidentListItem } from "./IncidentListItem";

/*
 * Sections de « Mes incidents ». Elles lisent la session ou l'URL : à rendre sous <Suspense>.
 */

export async function OngoingIncidents({
  searchParams,
}: {
  searchParams: Promise<{ ville?: string | string[] }>;
}) {
  await requireCitizenSession();
  const raw = (await searchParams).ville;
  const city = (Array.isArray(raw) ? raw[0] : raw)?.trim() || undefined;
  const incidents = await getOngoingIncidents(city);

  return (
    <>
      {/* Formulaire GET : fonctionne sans JavaScript et l'URL reste partageable. */}
      <form role="search" action={routes.profileIncidents} className="mb-4 flex gap-2">
        <label htmlFor="ville" className="sr-only">
          Filtrer par ville
        </label>
        <input
          id="ville"
          name="ville"
          type="search"
          defaultValue={city}
          placeholder="Filtrer par ville"
          autoComplete="address-level2"
          className="min-h-11 w-full min-w-0 rounded-full border border-border bg-background px-4 placeholder:text-muted"
        />
        <button type="submit" className={cn(buttonClasses("secondary"), "shrink-0")}>
          Filtrer
        </button>
      </form>
      {city && (
        <p className="mb-4 text-sm text-muted">
          {incidents.length} incident{incidents.length > 1 ? "s" : ""} en cours à {city} ·{" "}
          <Link href={routes.profileIncidents} className="text-accent hover:underline">
            Toutes les villes
          </Link>
        </p>
      )}
      {incidents.length === 0 ? (
        <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
          Aucun incident en cours{city ? ` à ${city}` : ""}. Si vous êtes témoin ou victime,
          déclarez-le ci-dessous.
        </p>
      ) : (
        <ul className="space-y-3">
          {incidents.map((incident) => (
            <li key={incident.id}>
              <IncidentListItem incident={incident} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

export async function DeclareIncidentSection() {
  await requireCitizenSession();
  const villains = await getVillains();
  return (
    <DeclareIncidentForm
      villains={villains.map(({ id, alias }) => ({ id, alias }))}
      action={declareIncidentAction}
    />
  );
}

export async function MyIncidents() {
  const token = await requireCitizenSession();
  const incidents = await getMyIncidents(token);

  if (incidents.length === 0) {
    return <p className="text-sm text-muted">Vous n&apos;avez encore déclaré aucun incident.</p>;
  }

  return (
    <ul className="grid gap-3 md:grid-cols-2">
      {incidents.map((incident) => (
        <li key={incident.id}>
          <IncidentListItem incident={incident} detailed />
        </li>
      ))}
    </ul>
  );
}

export function ListSkeleton({ items = 3 }: { items?: number }) {
  return (
    <div aria-hidden="true" className="space-y-3">
      {Array.from({ length: items }, (_, index) => (
        <div
          key={index}
          className="h-36 border-2 border-border bg-surface motion-safe:animate-pulse"
        />
      ))}
    </div>
  );
}
