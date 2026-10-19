import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { cn } from "@/components/ui/cn";
import { Container } from "@/components/ui/Container";
import { getPublishedVideos } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { routes } from "@/lib/routes";
import type { HeroVideo, SuperHeroSummary } from "@/types";

export const metadata: Metadata = {
  title: "Vidéothèque",
  description:
    "Les films consacrés aux super-héros de Heros-SOS et leurs vidéos de conseils, vérifiés par la modération.",
};

type VideosSearchParams = Promise<{ heros?: string | string[] }>;

function VideoCard({ video }: { video: HeroVideo }) {
  const titleId = `film-${video.id}-titre`;

  return (
    <article
      aria-labelledby={titleId}
      className="flex h-full flex-col border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5"
    >
      <p className="font-mono text-xs tracking-widest text-accent uppercase">
        {video.imdbId ? "Film" : "Vidéo"} · {video.hero.alias}
      </p>
      <h3 id={titleId} className="mt-2 font-display text-lg font-semibold">
        {video.title}
        {video.year && (
          <span className="font-mono text-sm font-normal text-muted"> ({video.year})</span>
        )}
      </h3>
      <p className="mt-2 text-sm text-pretty text-muted">{video.description}</p>
      <p className="mt-auto flex flex-wrap items-center justify-between gap-2 pt-4 text-sm">
        <span className="text-muted">
          Ajouté le <time dateTime={video.addedAt}>{formatDate(video.addedAt)}</time>
        </span>
        <a
          href={video.videoUrl}
          target="_blank"
          rel="noopener noreferrer nofollow"
          className="inline-flex min-h-11 items-center font-semibold text-accent hover:underline"
        >
          {video.imdbId ? "Voir sur IMDb" : "Regarder"}
          <span className="sr-only"> « {video.title} » (nouvel onglet)</span>
          <span aria-hidden="true">&nbsp;↗</span>
        </a>
      </p>
    </article>
  );
}

/** Lit l'URL (`?heros=`) : rendu sous <Suspense>. */
async function VideosGallery({ searchParams }: { searchParams: VideosSearchParams }) {
  const raw = (await searchParams).heros;
  const wanted = Array.isArray(raw) ? raw[0] : raw;
  const allVideos = await getPublishedVideos();

  // Filtres : uniquement les super-héros qui ont au moins un film publié.
  const heroes = [...new Map(allVideos.map(({ hero }) => [hero.id, hero] as const)).values()].sort(
    (a, b) => a.alias.localeCompare(b.alias, "fr"),
  );
  const current: SuperHeroSummary | undefined = heroes.find(({ id }) => id === wanted);
  const videos = current ? allVideos.filter(({ hero }) => hero.id === current.id) : allVideos;

  const filters = [
    { id: undefined, label: "Tous les héros" },
    ...heroes.map(({ id, alias }) => ({ id, label: alias })),
  ];

  return (
    <>
      <nav aria-label="Filtrer par super-héros" className="mt-6 mb-8">
        <ul className="flex flex-wrap gap-2">
          {filters.map((filter) => {
            const isCurrent = filter.id === current?.id;
            return (
              <li key={filter.id ?? "tous"}>
                <Link
                  href={filter.id ? `${routes.videos}?heros=${filter.id}` : routes.videos}
                  aria-current={isCurrent ? "page" : undefined}
                  className={cn(
                    "inline-flex min-h-10 items-center rounded-full border px-4 text-sm font-medium",
                    isCurrent
                      ? "border-primary bg-primary/15 text-foreground"
                      : "border-border text-muted hover:border-muted hover:text-foreground",
                  )}
                >
                  {filter.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </nav>

      <p role="status" className="mb-4 text-sm text-muted">
        {videos.length} film{videos.length > 1 ? "s" : ""}
        {current ? ` consacré${videos.length > 1 ? "s" : ""} à ${current.alias}` : ""}
      </p>

      {videos.length === 0 ? (
        <EmptyState>Aucun film publié pour l&apos;instant.</EmptyState>
      ) : (
        <ul className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
          {videos.map((video) => (
            <li key={video.id}>
              <VideoCard video={video} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

function GallerySkeleton() {
  return (
    <div aria-hidden="true" className="mt-6 grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
      {Array.from({ length: 3 }, (_, index) => (
        <div
          key={index}
          className="h-56 border-2 border-border bg-surface motion-safe:animate-pulse"
        />
      ))}
    </div>
  );
}

export default function VideosPage({ searchParams }: PageProps<"/videotheque">) {
  return (
    <Container className="py-10 sm:py-14">
      <h1 className="font-display text-3xl font-bold sm:text-4xl">Vidéothèque</h1>
      <p className="mt-2 max-w-2xl text-muted">
        Les films consacrés à vos super-héros et leurs vidéos de conseils. Chaque ajout est vérifié
        par l&apos;équipe de modération avant d&apos;être publié.
      </p>
      <Suspense fallback={<GallerySkeleton />}>
        <VideosGallery searchParams={searchParams} />
      </Suspense>
    </Container>
  );
}
