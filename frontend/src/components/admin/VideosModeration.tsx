import { deleteVideoAction, moderateVideoAction } from "@/app/actions/admin";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { FILM_QUERY_MAX_LENGTH, getHeroVideosPage, heroVideoKey } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { HeroVideo, ModerationStatus } from "@/types";
import { ConfirmDeleteButton } from "./ConfirmDeleteButton";
import { decisionsFor } from "./decisions";
import { ModerationBadge } from "./ModerationBadge";
import {
  EmptyState,
  ModerationFilter,
  readModerationFilter,
  type ModerationSearchParams,
} from "./ModerationFilter";
import { ModerationForm } from "./ModerationForm";
import { Pagination, readPageNumber } from "./Pagination";

const videoStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "À valider",
  valide: "Publié",
  rejete: "Refusé",
};

function VideoCard({ video }: { video: HeroVideo }) {
  const key = heroVideoKey(video);
  const titleId = `film-${video.hero.id}-${video.id}-titre`;

  return (
    <article
      aria-labelledby={titleId}
      className="grid gap-5 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto]"
    >
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-2">
          <ModerationBadge status={video.moderation} labels={videoStatusLabels} />
          <span className="text-sm text-muted">
            Vidéothèque de <span className="font-semibold text-foreground">{video.hero.alias}</span>
          </span>
        </div>
        <h2 id={titleId} className="mt-3 font-display text-lg font-semibold">
          {video.title}
        </h2>
        <p className="mt-1 text-sm text-pretty">{video.description}</p>
        <p className="mt-2 text-sm text-muted">
          Ajouté le{" "}
          <time dateTime={video.addedAt} className="font-mono">
            {formatDateTime(video.addedAt)}
          </time>{" "}
          ·{" "}
          <a
            href={video.videoUrl}
            target="_blank"
            rel="noopener noreferrer nofollow"
            className="break-all text-accent hover:underline"
          >
            Visionner<span className="sr-only"> « {video.title} » (nouvel onglet)</span>
          </a>
        </p>
      </div>
      <div className="flex flex-wrap items-start gap-2 lg:flex-col lg:items-end">
        <ModerationForm
          action={moderateVideoAction}
          id={key}
          subject={video.title}
          decisions={decisionsFor(video.moderation, {
            approve: "Publier",
            reject: "Refuser",
            restore: "Publier",
          })}
        />
        <ConfirmDeleteButton
          action={deleteVideoAction}
          id={key}
          subject={video.title}
          description={`Le film sera retiré de la vidéothèque de ${video.hero.alias}. Cette action est irréversible.`}
        />
      </div>
    </article>
  );
}

const PAGE_SIZE = 20;

/**
 * Paramètres d'URL de la liste : `?statut=…&titre=…&page=…`. La recherche s'appelle `titre` :
 * `q` et `heros` sont ceux du formulaire d'ajout affiché sur la même page.
 */
export type VideosModerationSearchParams = Promise<
  Awaited<ModerationSearchParams> & {
    titre?: string | string[];
    page?: string | string[];
  }
>;

/** Films ajoutés par les super-héros. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function VideosModeration({
  searchParams,
}: {
  searchParams: VideosModerationSearchParams;
}) {
  const token = await requireAdminSession();
  const filter = await readModerationFilter(searchParams);
  const pageNumber = await readPageNumber(searchParams);
  const rawTitle = (await searchParams).titre;
  const title = (Array.isArray(rawTitle) ? rawTitle[0] : rawTitle)?.trim() || undefined;
  const page = await getHeroVideosPage(
    token,
    { moderation: filter, query: title },
    { limit: PAGE_SIZE, offset: (pageNumber - 1) * PAGE_SIZE },
  );
  // Valeur de `?statut=` qui redonne ce filtre (absente pour le filtre par défaut).
  const status = filter === undefined ? "tous" : filter === "en-attente" ? undefined : filter;

  return (
    <>
      <ModerationFilter
        basePath={
          title
            ? `${routes.adminVideos}?${new URLSearchParams({ titre: title })}`
            : routes.adminVideos
        }
        current={filter}
        labels={{ "en-attente": "À valider", valide: "Publiés", rejete: "Refusés", tous: "Tous" }}
      />
      <form
        role="search"
        action={routes.adminVideos}
        className="mb-6 flex flex-wrap items-end gap-2"
      >
        {status && <input type="hidden" name="statut" value={status} />}
        <label className="text-sm font-medium">
          <span className="mb-2 block">Rechercher par titre</span>
          <input
            type="search"
            name="titre"
            defaultValue={title}
            maxLength={FILM_QUERY_MAX_LENGTH}
            placeholder="Extrait du titre"
            className="min-h-10 w-64 max-w-full border-2 border-border bg-background px-3 text-foreground placeholder:text-muted"
          />
        </label>
        <button type="submit" className={buttonClasses("secondary")}>
          Rechercher
        </button>
      </form>
      <p role="status" className="mb-4 text-sm text-muted">
        {page.total} film{page.total > 1 ? "s" : ""}
        {title ? ` pour « ${title} »` : ""}
      </p>
      {page.items.length === 0 ? (
        <EmptyState>
          {page.total === 0
            ? "Aucun film dans cette catégorie."
            : "Cette page n'existe pas : revenez à la première page."}
        </EmptyState>
      ) : (
        <>
          <ul className="space-y-4">
            {page.items.map((video) => (
              <li key={heroVideoKey(video)}>
                <VideoCard video={video} />
              </li>
            ))}
          </ul>
          <Pagination
            page={page}
            basePath={routes.adminVideos}
            query={{ statut: status, titre: title }}
          />
        </>
      )}
    </>
  );
}
