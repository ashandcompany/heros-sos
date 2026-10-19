import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense, type ReactNode } from "react";
import { Avatar } from "@/components/ui/Avatar";
import { Badge } from "@/components/ui/Badge";
import { Container } from "@/components/ui/Container";
import { RatingStars } from "@/components/ui/RatingStars";
import {
  getHeroPublicReviews,
  getPublishedVideos,
  getResolvedIncidentsByHero,
  getSuperHero,
} from "@/lib/api";
import { dossierNumber } from "@/lib/dossier";
import { formatDate, formatInteger, formatLocation, incidentTypeLabels } from "@/lib/format";
import { routes, villainSheetRoute } from "@/lib/routes";

export async function generateMetadata({
  params,
}: PageProps<"/encyclopedie/super-heros/[id]">): Promise<Metadata> {
  const hero = await getSuperHero((await params).id);
  return hero
    ? { title: hero.alias, description: `${hero.alias} : ${hero.power}.` }
    : { title: "Super-héros introuvable" };
}

function Section({ id, title, children }: { id: string; title: string; children: ReactNode }) {
  return (
    <section aria-labelledby={id}>
      <h2 id={id} className="mb-4 font-display text-xl font-semibold">
        {title}
      </h2>
      {children}
    </section>
  );
}

function Empty({ children }: { children: ReactNode }) {
  return (
    <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
      {children}
    </p>
  );
}

/** Lit les paramètres d'URL : rendu sous <Suspense>. */
async function HeroSheet({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const hero = await getSuperHero(id);
  if (!hero) notFound();

  const [interventions, videos, reviews] = await Promise.all([
    getResolvedIncidentsByHero(hero.id),
    getPublishedVideos(hero.id),
    getHeroPublicReviews(hero.id),
  ]);

  // Vilains déjà neutralisés par ce héros, parmi ses dernières interventions.
  const enemies = [
    ...new Map(
      interventions.flatMap(({ villain }) => (villain ? [[villain.id, villain] as const] : [])),
    ).values(),
  ];

  return (
    <div className="space-y-12">
      <header className="panel paper flex flex-col items-center gap-6 p-6 text-center sm:flex-row sm:items-start sm:p-8 sm:text-left">
        <div className="shrink-0 border-4 border-background bg-background shadow-[3px_3px_0_0] shadow-black/40">
          <Avatar name={hero.alias} src={hero.avatarUrl} size={128} square />
        </div>
        <div className="min-w-0 flex-1">
          <p className="font-mono text-xs tracking-widest text-ink/75 uppercase">
            Matricule {dossierNumber(hero.id, "AG")}
          </p>
          <h1 className="mt-1 font-display text-3xl font-bold tracking-wider uppercase sm:text-4xl">
            {hero.alias}
          </h1>
          <p className="mt-2 text-lg text-pretty text-ink/85">{hero.power}</p>
          <p className="mt-3 inline-flex items-center gap-2 border-2 border-sage bg-sage px-2 py-0.5 font-mono text-xs font-medium tracking-widest text-paper uppercase">
            <span aria-hidden="true">✓</span> Agent vérifié
          </p>
          <dl className="mt-6 grid grid-cols-2 gap-4 sm:max-w-sm">
            <div>
              <dt className="font-mono text-xs tracking-widest text-ink/75 uppercase">Note</dt>
              <dd className="mt-1">
                <RatingStars rating={hero.averageRating} />
              </dd>
            </div>
            <div>
              <dt className="font-mono text-xs tracking-widest text-ink/75 uppercase">
                Interventions
              </dt>
              <dd className="mt-1 digital text-xl">{formatInteger(hero.interventionsCount)}</dd>
            </div>
          </dl>
        </div>
      </header>

      <div className="grid gap-12 lg:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)]">
        <div className="space-y-12">
          <Section id="interventions-title" title="Dernières interventions réussies">
            {interventions.length === 0 ? (
              <Empty>Aucune intervention publiée pour l&apos;instant.</Empty>
            ) : (
              <ul className="space-y-3">
                {interventions.map((incident) => (
                  <li
                    key={incident.id}
                    className="border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-4 sm:p-5"
                  >
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge tone="success">Résolu</Badge>
                      <span className="text-xs text-muted">
                        {incidentTypeLabels[incident.type]}
                      </span>
                    </div>
                    <h3 className="mt-2 font-display font-semibold">{incident.title}</h3>
                    <p className="mt-1 text-sm text-muted">
                      {formatLocation(incident.location)} · le{" "}
                      <time dateTime={incident.resolvedAt}>{formatDate(incident.resolvedAt)}</time>
                    </p>
                  </li>
                ))}
              </ul>
            )}
          </Section>

          <Section id="reviews-title" title="Avis des citoyens">
            {reviews.length === 0 ? (
              <Empty>Aucun avis publié pour l&apos;instant.</Empty>
            ) : (
              <ul className="space-y-4">
                {reviews.map((review) => (
                  <li
                    key={review.incidentId}
                    className="border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <RatingStars rating={review.rating} />
                      <time dateTime={review.ratedAt} className="font-mono text-xs text-muted">
                        {formatDate(review.ratedAt)}
                      </time>
                    </div>
                    <blockquote className="mt-3 border-l-2 border-border pl-3 text-pretty">
                      {review.comment}
                    </blockquote>
                    {review.reply && (
                      <div className="mt-4 rounded-xl border border-border bg-background p-4">
                        <p className="text-sm font-semibold">Réponse de {hero.alias}</p>
                        <p className="mt-1 text-sm text-pretty">{review.reply.content}</p>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            )}
          </Section>
        </div>

        <div className="space-y-12">
          <Section id="enemies-title" title="Vilains neutralisés">
            {enemies.length === 0 ? (
              <Empty>Aucun vilain identifié lors de ses dernières interventions.</Empty>
            ) : (
              <ul className="flex flex-wrap gap-2">
                {enemies.map((villain) => (
                  <li key={villain.id}>
                    <Link
                      href={villainSheetRoute(villain.id)}
                      className="inline-flex min-h-11 items-center rounded-full border border-border px-4 text-sm font-medium hover:border-muted hover:text-accent"
                    >
                      {villain.alias}
                    </Link>
                  </li>
                ))}
              </ul>
            )}
          </Section>

          <Section id="videos-title" title="Vidéothèque">
            {videos.length === 0 ? (
              <Empty>Aucun film publié pour l&apos;instant.</Empty>
            ) : (
              <>
                <ul className="space-y-3">
                  {videos.map((video) => (
                    <li
                      key={video.id}
                      className="border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-4"
                    >
                      <p className="font-display font-semibold">
                        {video.title}
                        {video.year && (
                          <span className="font-mono text-sm font-normal text-muted">
                            {" "}
                            ({video.year})
                          </span>
                        )}
                      </p>
                      <a
                        href={video.videoUrl}
                        target="_blank"
                        rel="noopener noreferrer nofollow"
                        className="mt-1 inline-flex min-h-11 items-center text-sm font-semibold text-accent hover:underline"
                      >
                        {video.imdbId ? "Voir sur IMDb" : "Regarder"}
                        <span className="sr-only"> « {video.title} » (nouvel onglet)</span>
                        <span aria-hidden="true">&nbsp;↗</span>
                      </a>
                    </li>
                  ))}
                </ul>
                <p className="mt-3 text-sm">
                  <Link
                    href={`${routes.videos}?heros=${hero.id}`}
                    className="text-accent hover:underline"
                  >
                    Tous ses films dans la vidéothèque
                  </Link>
                </p>
              </>
            )}
          </Section>
        </div>
      </div>
    </div>
  );
}

export default function HeroSheetPage({ params }: PageProps<"/encyclopedie/super-heros/[id]">) {
  return (
    <Container className="py-10 sm:py-14">
      <p className="mb-6 text-sm">
        <Link href={`${routes.encyclopedia}#super-heros`} className="text-accent hover:underline">
          ← Tous les super-héros
        </Link>
      </p>
      <Suspense
        fallback={
          <div aria-hidden="true" className="panel h-72 opacity-40 motion-safe:animate-pulse" />
        }
      >
        <HeroSheet params={params} />
      </Suspense>
    </Container>
  );
}
