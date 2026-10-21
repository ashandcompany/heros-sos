import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { StatusBadge } from "@/components/hero/IncidentBadges";
import { threatTone } from "@/components/encyclopedia/VillainCard";
import { VillainSpecialties } from "@/components/encyclopedia/VillainSpecialties";
import { Badge } from "@/components/ui/Badge";
import { Container } from "@/components/ui/Container";
import { getIncidentsByVillain, getVillain } from "@/lib/api";
import { dossierNumber } from "@/lib/dossier";
import {
  formatDate,
  formatInteger,
  formatLocation,
  threatLevelLabels,
  villainStatusLabels,
} from "@/lib/format";
import { heroSheetRoute, routes } from "@/lib/routes";

export async function generateMetadata({
  params,
}: PageProps<"/encyclopedie/super-vilains/[id]">): Promise<Metadata> {
  const villain = await getVillain((await params).id);
  return villain
    ? { title: villain.alias, description: villain.description }
    : { title: "Super-vilain introuvable" };
}

/** Lit les paramètres d'URL : rendu sous <Suspense>. */
async function VillainSheet({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const villain = await getVillain(id);
  if (!villain) notFound();

  const incidents = await getIncidentsByVillain(villain.id);
  // Super-héros intervenus sur ses derniers méfaits.
  const opponents = [
    ...new Map(incidents.flatMap(({ hero }) => (hero ? [[hero.id, hero] as const] : []))).values(),
  ];

  return (
    <div className="space-y-12">
      <header className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
        <p className="font-mono text-xs tracking-widest text-muted uppercase">
          Dossier {dossierNumber(villain.id, "SV")}
        </p>
        <h1 className="mt-1 font-display text-3xl font-bold sm:text-4xl">{villain.alias}</h1>
        <div className="mt-3 flex flex-wrap gap-2">
          <Badge tone={threatTone[villain.threatLevel]}>
            Menace {threatLevelLabels[villain.threatLevel].toLowerCase()}
          </Badge>
          <Badge tone={villain.status === "emprisonne" ? "sage" : "warning"}>
            {villainStatusLabels[villain.status]}
          </Badge>
        </div>
        <p className="mt-4 max-w-2xl text-lg text-pretty">{villain.description}</p>
        {villain.specialties.length > 0 && (
          <div className="mt-5">
            <h2 className="mb-2 text-sm font-medium text-muted">Spécialités</h2>
            <VillainSpecialties specialties={villain.specialties} />
          </div>
        )}
        <p className="mt-6 text-sm">
          <span className="font-mono text-2xl font-semibold">
            {formatInteger(villain.reportedIncidentsCount)}
          </span>{" "}
          <span className="text-muted">incidents signalés</span>
        </p>
        {villain.status === "en-liberte" && (
          <p className="mt-6 rounded-xl border border-rating/40 bg-rating/10 px-4 py-3 text-sm">
            <span className="font-semibold text-rating">Vous pensez l&apos;avoir vu&nbsp;?</span> Ne
            l&apos;approchez pas : éloignez-vous puis{" "}
            <Link href={routes.profileIncidents} className="text-accent underline">
              déclarez l&apos;incident
            </Link>
            .
          </p>
        )}
      </header>

      <div className="grid gap-12 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
        <section aria-labelledby="incidents-title">
          <h2 id="incidents-title" className="mb-4 font-display text-xl font-semibold">
            Derniers méfaits
          </h2>
          {incidents.length === 0 ? (
            <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
              Aucun incident publié pour l&apos;instant.
            </p>
          ) : (
            <ul className="space-y-3">
              {incidents.map((incident) => (
                <li
                  key={incident.id}
                  className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-4 sm:p-5"
                >
                  <div className="flex flex-wrap items-center gap-2">
                    <StatusBadge status={incident.status} />
                    <span className="text-xs text-muted">{incident.type.label}</span>
                  </div>
                  <h3 className="mt-2 font-display font-semibold">{incident.title}</h3>
                  <p className="mt-1 text-sm text-muted">
                    {formatLocation(incident.location)} · signalé le{" "}
                    <time dateTime={incident.reportedAt}>{formatDate(incident.reportedAt)}</time>
                  </p>
                  {incident.hero && (
                    <p className="mt-2 text-sm">
                      <span className="text-muted">Super-héros : </span>
                      <Link
                        href={heroSheetRoute(incident.hero.id)}
                        className="font-medium text-accent hover:underline"
                      >
                        {incident.hero.alias}
                      </Link>
                    </p>
                  )}
                </li>
              ))}
            </ul>
          )}
        </section>

        <section aria-labelledby="opponents-title">
          <h2 id="opponents-title" className="mb-4 font-display text-xl font-semibold">
            Ils l&apos;ont affronté
          </h2>
          {opponents.length === 0 ? (
            <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
              Aucun super-héros ne l&apos;a encore affronté.
            </p>
          ) : (
            <ul className="flex flex-wrap gap-2">
              {opponents.map((hero) => (
                <li key={hero.id}>
                  <Link
                    href={heroSheetRoute(hero.id)}
                    className="inline-flex min-h-11 items-center rounded-full border border-border px-4 text-sm font-medium hover:border-muted hover:text-accent"
                  >
                    {hero.alias}
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </section>
      </div>
    </div>
  );
}

export default function VillainSheetPage({
  params,
}: PageProps<"/encyclopedie/super-vilains/[id]">) {
  return (
    <Container className="py-10 sm:py-14">
      <p className="mb-6 text-sm">
        <Link href={`${routes.encyclopedia}#super-vilains`} className="text-accent hover:underline">
          ← Tous les super-vilains
        </Link>
      </p>
      <Suspense
        fallback={
          <div
            aria-hidden="true"
            className="h-72 border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        }
      >
        <VillainSheet params={params} />
      </Suspense>
    </Container>
  );
}
