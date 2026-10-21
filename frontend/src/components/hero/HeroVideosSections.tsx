import { deleteMyVideoAction } from "@/app/actions/hero";
import { ConfirmDeleteButton } from "@/components/admin/ConfirmDeleteButton";
import { ModerationBadge } from "@/components/admin/ModerationBadge";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { getMyVideos, searchImdbTitles } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireHeroSession } from "@/lib/server-session";
import { apiFailureMessage } from "@/lib/validation/form-state";
import type { ModerationStatus } from "@/types";
import { AddVideoForm } from "./AddVideoForm";

/*
 * Sections de « Ma vidéothèque ». Elles lisent la session ou l'URL : à rendre sous <Suspense>.
 */

const videoStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "En cours de validation",
  valide: "Publié",
  rejete: "Refusé par la modération",
};

export type VideoSearchParams = Promise<{ q?: string | string[] }>;

export async function ImdbSearch({ searchParams }: { searchParams: VideoSearchParams }) {
  const { token } = await requireHeroSession();
  const raw = (await searchParams).q;
  const query = (Array.isArray(raw) ? raw[0] : raw)?.trim() ?? "";
  const [search, videos] = await Promise.all([
    query ? searchImdbTitles(token, query) : null,
    getMyVideos(token),
  ]);
  const owned = new Set(videos.map(({ imdbId }) => imdbId));
  const results = search?.ok ? search.titles : [];

  return (
    <>
      {/* Formulaire GET : fonctionne sans JavaScript et l'URL reste partageable. */}
      <form role="search" action={routes.heroVideos} className="mb-4 flex gap-2">
        <label htmlFor="q" className="sr-only">
          Titre du film
        </label>
        <input
          id="q"
          name="q"
          type="search"
          defaultValue={query}
          placeholder="Titre du film"
          className="min-h-11 w-full min-w-0 rounded-full border border-border bg-background px-4 placeholder:text-muted"
        />
        <button type="submit" className={cn(buttonClasses("secondary"), "shrink-0")}>
          Rechercher
        </button>
      </form>

      {search &&
        (!search.ok ? (
          <p role="alert" className="text-sm text-danger">
            {search.error === "catalog-unavailable"
              ? "Le catalogue IMDb est momentanément indisponible. Réessayez dans quelques instants."
              : apiFailureMessage(search)}
          </p>
        ) : results.length === 0 ? (
          <EmptyState>Aucun film ne correspond à «&nbsp;{query}&nbsp;» sur IMDb.</EmptyState>
        ) : (
          <>
            <p role="status" className="mb-3 text-sm text-muted">
              {results.length} résultat{results.length > 1 ? "s" : ""} sur IMDb
            </p>
            <ul className="space-y-3">
              {results.map((title) => (
                <li
                  key={title.id}
                  className="flex flex-wrap items-start justify-between gap-4 rounded-2xl border border-border bg-background p-4"
                >
                  <div className="min-w-0 flex-1">
                    <p className="font-display font-semibold">
                      {title.title}
                      {title.year && (
                        <span className="font-mono text-sm text-muted"> ({title.year})</span>
                      )}
                    </p>
                    {title.plot && (
                      <p className="mt-1 text-sm text-pretty text-muted">{title.plot}</p>
                    )}
                  </div>
                  {owned.has(title.id) ? (
                    <p className="text-sm text-muted">Déjà dans votre vidéothèque</p>
                  ) : (
                    <AddVideoForm imdbId={title.id} title={title.title} />
                  )}
                </li>
              ))}
            </ul>
          </>
        ))}
    </>
  );
}

export async function MyVideos() {
  const { token } = await requireHeroSession();
  const videos = await getMyVideos(token);

  if (videos.length === 0) {
    return (
      <EmptyState>
        Votre vidéothèque est vide. Recherchez un film sur IMDb pour l&apos;ajouter.
      </EmptyState>
    );
  }

  return (
    <ul className="space-y-4">
      {videos.map((video) => {
        const titleId = `film-${video.id}-titre`;
        return (
          <li key={video.id}>
            <article
              aria-labelledby={titleId}
              className="flex flex-wrap items-start justify-between gap-4 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5"
            >
              <div className="min-w-0 flex-1">
                <ModerationBadge status={video.moderation} labels={videoStatusLabels} />
                <h3 id={titleId} className="mt-3 font-display text-lg font-semibold">
                  {video.title}
                  {video.year && (
                    <span className="font-mono text-sm font-normal text-muted">
                      {" "}
                      ({video.year})
                    </span>
                  )}
                </h3>
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
                    className="text-accent hover:underline"
                  >
                    {video.imdbId ? "Fiche IMDb" : "Visionner"}
                    <span className="sr-only"> « {video.title} » (nouvel onglet)</span>
                  </a>
                </p>
              </div>
              <ConfirmDeleteButton
                action={deleteMyVideoAction}
                id={video.id}
                subject={video.title}
                description="Le film sera retiré de votre vidéothèque. Pour le rajouter, il devra de nouveau être validé par la modération."
              />
            </article>
          </li>
        );
      })}
    </ul>
  );
}
