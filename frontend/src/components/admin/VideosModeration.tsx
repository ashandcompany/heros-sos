import { deleteVideoAction, moderateVideoAction } from "@/app/actions/admin";
import { getHeroVideos, heroVideoKey } from "@/lib/api";
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

/** Films ajoutés par les super-héros. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function VideosModeration({ searchParams }: { searchParams: ModerationSearchParams }) {
  const token = await requireAdminSession();
  const filter = await readModerationFilter(searchParams);
  const videos = await getHeroVideos(token, filter);

  return (
    <>
      <ModerationFilter
        basePath={routes.adminVideos}
        current={filter}
        labels={{ "en-attente": "À valider", valide: "Publiés", rejete: "Refusés", tous: "Tous" }}
      />
      {videos.length === 0 ? (
        <EmptyState>Aucun film dans cette catégorie.</EmptyState>
      ) : (
        <ul className="space-y-4">
          {videos.map((video) => (
            <li key={heroVideoKey(video)}>
              <VideoCard video={video} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
