import type { Metadata } from "next";
import Image from "next/image";
import Link from "next/link";
import { Suspense } from "react";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { Pagination, readPageNumber } from "@/components/admin/Pagination";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { Container } from "@/components/ui/Container";
import { FILM_QUERY_MAX_LENGTH, getFilmHeroes, getPublishedFilmsPage } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { routes } from "@/lib/routes";
import type { FilmHero, PublicFilm, SuperHeroSummary } from "@/types";

export const metadata: Metadata = {
  title: "Vidéothèque",
  description:
    "Les films consacrés aux super-héros de Heros-SOS et leurs vidéos de conseils, vérifiés par la modération.",
};

type VideosSearchParams = PageProps<"/videotheque">["searchParams"];

/** Multiple de 2 et de 3 : les pages pleines remplissent la grille à toutes les largeurs. */
const PAGE_SIZE = 12;
const GALLERY_ANCHOR = "films";
/** Héros nommés sur une carte, les suivants sont comptés (« +3 »). */
const CARD_HEROES = 3;

const firstValue = (value: string | string[] | undefined) =>
  (Array.isArray(value) ? value[0] : value)?.trim() || undefined;

/** Vidéothèque avec ces filtres, à la première page. */
function filteredHref(query: Record<string, string | undefined>): string {
  const params = new URLSearchParams();
  for (const [name, value] of Object.entries(query)) if (value) params.set(name, value);
  const search = params.toString();
  return search ? `${routes.videos}?${search}` : routes.videos;
}

/** Colonne de l'affiche : au moins au format 2:3, étirée à la hauteur de la carte. */
const POSTER_COLUMN = "relative min-h-48 w-32 shrink-0 border-r-2 border-border bg-background";

/**
 * Affiche du film, décorative (alt vide) : le titre est juste à côté. Une vidéo libre n'en a pas,
 * un cadre la remplace pour garder les cartes alignées.
 */
function FilmPoster({ src }: { src: string | undefined }) {
  if (!src) {
    return (
      <div
        aria-hidden="true"
        className={`${POSTER_COLUMN} flex items-center justify-center text-3xl text-muted`}
      >
        ▶
      </div>
    );
  }
  return (
    <div className={POSTER_COLUMN}>
      <Image
        src={src}
        alt=""
        fill
        sizes="8rem"
        // Affiche hébergée par le catalogue (OMDb / IMDb), déjà redimensionnée : servie telle quelle
        // plutôt que d'autoriser cet hôte dans l'optimiseur de Next.
        unoptimized
        className="object-cover"
      />
    </div>
  );
}

function FilmCard({ film }: { film: PublicFilm }) {
  const titleId = `film-${film.id}-titre`;
  const shown = film.heroes.slice(0, CARD_HEROES);
  const hidden = film.heroes.slice(CARD_HEROES);

  return (
    <article
      aria-labelledby={titleId}
      className="flex h-full overflow-hidden border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)]"
    >
      <FilmPoster src={film.posterUrl} />
      <div className="flex min-w-0 flex-1 flex-col p-4">
        <p className="font-mono text-xs tracking-widest text-accent uppercase">
          {film.imdbId ? "Film" : "Vidéo"} · {shown.map(({ alias }) => alias).join(", ")}
          {hidden.length > 0 && (
            <>
              <span aria-hidden="true" title={hidden.map(({ alias }) => alias).join(", ")}>
                {" "}
                +{hidden.length}
              </span>
              <span className="sr-only">, {hidden.map(({ alias }) => alias).join(", ")}</span>
            </>
          )}
        </p>
        <h3
          id={titleId}
          className="mt-1 font-display text-lg leading-snug font-semibold text-balance"
        >
          {film.title}
          {film.year && (
            <span className="font-mono text-sm font-normal text-muted"> ({film.year})</span>
          )}
        </h3>
        <p className="mt-2 mb-3 line-clamp-4 text-sm text-pretty text-muted">{film.description}</p>
        <div className="mt-auto flex flex-wrap items-end justify-between gap-x-3 border-t border-border/60 pt-2 text-sm">
          <span className="py-3 text-xs text-muted">
            Ajouté le <time dateTime={film.addedAt}>{formatDate(film.addedAt)}</time>
          </span>
          <a
            href={film.videoUrl}
            target="_blank"
            rel="noopener noreferrer nofollow"
            className="inline-flex min-h-11 items-center font-semibold text-accent hover:underline"
          >
            {film.imdbId ? "Voir sur IMDb" : "Regarder"}
            <span className="sr-only"> « {film.title} » (nouvel onglet)</span>
            <span aria-hidden="true">&nbsp;↗</span>
          </a>
        </div>
      </div>
    </article>
  );
}

/**
 * Filtre (GET : fonctionne sans JavaScript et l'URL reste partageable). Une liste déroulante
 * plutôt que des onglets : elle reste utilisable quel que soit le nombre de héros.
 */
function GalleryFilter({
  heroes,
  current,
  query,
}: {
  heroes: FilmHero[];
  current: SuperHeroSummary | undefined;
  query: string | undefined;
}) {
  const fieldClasses =
    "min-h-11 w-full min-w-0 border-2 border-border bg-background px-3 text-foreground placeholder:text-muted";

  return (
    <form
      role="search"
      action={routes.videos}
      className="mt-6 mb-8 grid gap-3 sm:grid-cols-[minmax(0,1fr)_minmax(0,16rem)_auto] sm:items-end"
    >
      <label className="text-sm font-medium">
        <span className="mb-2 block">Titre du film</span>
        <input
          type="search"
          name="q"
          defaultValue={query}
          maxLength={FILM_QUERY_MAX_LENGTH}
          placeholder="Ex. : Iron Man"
          className={fieldClasses}
        />
      </label>
      <label className="text-sm font-medium">
        <span className="mb-2 block">Super-héros</span>
        <select name="heros" defaultValue={current?.id ?? ""} className={fieldClasses}>
          <option value="">Tous les héros</option>
          {heroes.map(({ id, alias, filmCount }) => (
            <option key={id} value={id}>
              {alias} ({filmCount})
            </option>
          ))}
        </select>
      </label>
      <div className="flex flex-wrap items-center gap-3">
        <button type="submit" className={buttonClasses("secondary")}>
          Filtrer
        </button>
        {(current || query) && (
          <Link href={routes.videos} className="text-sm text-accent hover:underline">
            Réinitialiser
          </Link>
        )}
      </div>
    </form>
  );
}

/** Lit l'URL (`?heros=`, `?q=`, `?page=`) : rendu sous <Suspense>. */
async function VideosGallery({ searchParams }: { searchParams: VideosSearchParams }) {
  const params = await searchParams;
  const query = firstValue(params.q);
  const pageNumber = await readPageNumber(searchParams);
  const heroes = await getFilmHeroes();
  // Héros absent de la liste (lien ancien) : filtre ignoré.
  const current = heroes.find(({ id }) => id === firstValue(params.heros));
  const page = await getPublishedFilmsPage(
    { heroId: current?.id, query },
    { limit: PAGE_SIZE, offset: (pageNumber - 1) * PAGE_SIZE },
  );
  const filterQuery = { heros: current?.id, q: query };

  if (heroes.length === 0) {
    return (
      <div className="mt-6">
        <EmptyState>Aucun film publié pour l&apos;instant.</EmptyState>
      </div>
    );
  }

  return (
    <>
      <GalleryFilter heroes={heroes} current={current} query={query} />

      <p id={GALLERY_ANCHOR} role="status" className="mb-4 scroll-mt-20 text-sm text-muted">
        {page.total} film{page.total > 1 ? "s" : ""}
        {current ? ` consacré${page.total > 1 ? "s" : ""} à ${current.alias}` : ""}
        {query ? ` pour « ${query} »` : ""}
      </p>

      {page.total === 0 ? (
        <EmptyState>Aucun film ne correspond à cette recherche.</EmptyState>
      ) : page.items.length === 0 ? (
        // Page au-delà de la dernière (lien ancien ou saisi à la main).
        <EmptyState>
          Cette page n&apos;existe pas.{" "}
          <Link href={filteredHref(filterQuery)} className="text-accent hover:underline">
            Revenir aux premiers films
          </Link>
        </EmptyState>
      ) : (
        <>
          <ul className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
            {page.items.map((film) => (
              <li key={film.id}>
                <FilmCard film={film} />
              </li>
            ))}
          </ul>
          <Pagination
            page={page}
            basePath={routes.videos}
            query={filterQuery}
            anchor={GALLERY_ANCHOR}
          />
        </>
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
          className="flex h-56 border-2 border-border bg-surface motion-safe:animate-pulse"
        >
          <div className="w-32 shrink-0 border-r-2 border-border bg-background" />
        </div>
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
