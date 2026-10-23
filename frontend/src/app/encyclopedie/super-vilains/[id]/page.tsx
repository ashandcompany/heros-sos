import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { StatusBadge } from "@/components/hero/IncidentBadges";
import { ThreatMeter } from "@/components/encyclopedia/ThreatMeter";
import { VillainStatusStamp } from "@/components/encyclopedia/VillainCard";
import { VillainMugshot } from "@/components/encyclopedia/VillainMugshot";
import { VillainSpecialties } from "@/components/encyclopedia/VillainSpecialties";
import { Container } from "@/components/ui/Container";
import { getIncidentsByVillain, getVillain, getVillainFilms } from "@/lib/api";
import { dossierNumber } from "@/lib/dossier";
import { formatDate, formatInteger, formatLocation } from "@/lib/format";
import { encyclopediaVillainsRoute, heroSheetRoute, routes, villainSheetRoute } from "@/lib/routes";
import { getSiteUrl } from "@/lib/site";

export async function generateMetadata({
  params,
}: PageProps<"/encyclopedie/super-vilains/[id]">): Promise<Metadata> {
  const villain = await getVillain((await params).id);
  if (!villain) return { title: "Super-vilain introuvable" };

  // Rendu à la requête (paramètre d'URL) : SITE_URL n'est pas figée au build.
  return {
    metadataBase: new URL(getSiteUrl()),
    title: villain.alias,
    description: villain.description,
    alternates: { canonical: villainSheetRoute(villain.id) },
    openGraph: { type: "profile", title: villain.alias, description: villain.description },
  };
}

/** Lit les paramètres d'URL : rendu sous <Suspense>. */
async function VillainSheet({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const villain = await getVillain(id);
  if (!villain) notFound();

  const [incidents, films] = await Promise.all([
    getIncidentsByVillain(villain.id),
    getVillainFilms(villain.id),
  ]);
  // Super-héros intervenus sur ses derniers méfaits.
  const opponents = [
    ...new Map(incidents.flatMap(({ hero }) => (hero ? [[hero.id, hero] as const] : []))).values(),
  ];

  return (
    <div className="space-y-12">
      <header className="panel paper paper--wanted flex flex-col items-center gap-6 p-6 text-center sm:flex-row sm:items-start sm:p-8 sm:text-left">
        <VillainMugshot alias={villain.alias} size={128} />
        <div className="min-w-0 flex-1">
          <div className="flex flex-col items-center gap-3 sm:flex-row sm:items-start sm:justify-between">
            <div className="min-w-0">
              <p className="font-mono text-xs tracking-widest text-ink/75 uppercase">
                Dossier {dossierNumber(villain.id, "SV")}
              </p>
              <h1 className="mt-1 font-display text-3xl font-bold tracking-wider uppercase sm:text-4xl">
                {villain.alias}
              </h1>
            </div>
            <VillainStatusStamp status={villain.status} className="mt-1 sm:mt-3" />
          </div>
          <p className="mt-3 max-w-2xl text-lg text-pretty text-ink/85">{villain.description}</p>

          <dl className="mt-6 grid grid-cols-2 gap-4 text-left sm:max-w-md">
            <div>
              <dt className="font-mono text-xs tracking-widest text-ink/75 uppercase">Menace</dt>
              <dd className="mt-2">
                <ThreatMeter level={villain.threatLevel} />
              </dd>
            </div>
            <div>
              <dt className="font-mono text-xs tracking-widest text-ink/75 uppercase">
                Incidents signalés
              </dt>
              <dd className="mt-1">
                <span className="digital text-xl">
                  {formatInteger(villain.reportedIncidentsCount)}
                </span>
              </dd>
            </div>
          </dl>

          {villain.specialties.length > 0 && (
            <div className="mt-6">
              <h2 className="mb-2 font-mono text-xs tracking-widest text-ink/75 uppercase">
                Spécialités
              </h2>
              <VillainSpecialties
                specialties={villain.specialties}
                className="justify-center sm:justify-start"
              />
            </div>
          )}

          {villain.status === "en-liberte" && (
            <p className="mt-6 border-2 border-brick bg-brick/10 px-4 py-3 text-left text-sm text-pretty">
              <span className="font-semibold text-brick">Vous pensez l&apos;avoir vu&nbsp;?</span>{" "}
              Ne l&apos;approchez pas : éloignez-vous puis{" "}
              <Link
                href={routes.profileIncidents}
                className="font-semibold text-primary-ink underline"
              >
                déclarez l&apos;incident
              </Link>
              .
            </p>
          )}
        </div>
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

          <h2 id="films-title" className="mt-12 mb-4 font-display text-xl font-semibold">
            Au cinéma
          </h2>
          {films.length === 0 ? (
            <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
              Aucun film répertorié pour l&apos;instant.
            </p>
          ) : (
            <ul aria-labelledby="films-title" className="space-y-3">
              {films.map((film) => (
                <li
                  key={film.id}
                  className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-4"
                >
                  <p className="font-display font-semibold">
                    {film.title}
                    {film.year && (
                      <span className="font-mono text-sm font-normal text-muted">
                        {" "}
                        ({film.year})
                      </span>
                    )}
                  </p>
                  {film.videoUrl && (
                    <a
                      href={film.videoUrl}
                      target="_blank"
                      rel="noopener noreferrer nofollow"
                      className="mt-1 inline-flex min-h-11 items-center text-sm font-semibold text-accent hover:underline"
                    >
                      {film.imdbId ? "Voir sur IMDb" : "Regarder"}
                      <span className="sr-only"> « {film.title} » (nouvel onglet)</span>
                      <span aria-hidden="true">&nbsp;↗</span>
                    </a>
                  )}
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
        <Link href={encyclopediaVillainsRoute} className="text-accent hover:underline">
          ← Tous les super-vilains
        </Link>
      </p>
      <Suspense
        fallback={
          <div aria-hidden="true" className="panel h-72 opacity-40 motion-safe:animate-pulse" />
        }
      >
        <VillainSheet params={params} />
      </Suspense>
    </Container>
  );
}
