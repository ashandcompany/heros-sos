import { moderateCommentAction } from "@/app/actions/admin";
import { RatingStars } from "@/components/ui/RatingStars";
import { getRatingsForModeration } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { ModerationStatus, RatingForModeration } from "@/types";
import { decisionsFor } from "./decisions";
import { ModerationBadge } from "./ModerationBadge";
import {
  EmptyState,
  ModerationFilter,
  readModerationFilter,
  type ModerationSearchParams,
} from "./ModerationFilter";
import { ModerationForm } from "./ModerationForm";

const commentStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "À modérer",
  valide: "Publié",
  rejete: "Masqué",
};

function CommentCard({ rating }: { rating: RatingForModeration }) {
  const titleId = `avis-${rating.incidentId}-titre`;

  return (
    <article
      aria-labelledby={titleId}
      className="grid gap-5 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto]"
    >
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-3">
          <ModerationBadge status={rating.commentModeration} labels={commentStatusLabels} />
          <RatingStars rating={rating.rating} />
        </div>
        <h2 id={titleId} className="mt-3 font-display font-semibold">
          Avis sur {rating.hero.alias}
          <span className="font-sans font-normal text-muted"> · {rating.incident.title}</span>
        </h2>
        <blockquote className="mt-2 border-l-2 border-border pl-3 text-sm text-pretty">
          {rating.comment}
        </blockquote>
        <p className="mt-2 text-xs text-muted">
          Envoyé le{" "}
          <time dateTime={rating.ratedAt} className="font-mono">
            {formatDateTime(rating.ratedAt)}
          </time>
        </p>
      </div>
      <ModerationForm
        action={moderateCommentAction}
        id={rating.incidentId}
        subject={`avis sur ${rating.hero.alias}`}
        decisions={decisionsFor(rating.commentModeration, {
          approve: "Publier",
          reject: "Masquer",
          restore: "Publier",
        })}
      />
    </article>
  );
}

/** Commentaires des évaluations. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function CommentsModeration({
  searchParams,
}: {
  searchParams: ModerationSearchParams;
}) {
  const token = await requireAdminSession();
  const filter = await readModerationFilter(searchParams);
  const ratings = await getRatingsForModeration(token, filter);

  return (
    <>
      <ModerationFilter
        basePath={routes.adminComments}
        current={filter}
        labels={{ "en-attente": "À modérer", valide: "Publiés", rejete: "Masqués", tous: "Tous" }}
      />
      {ratings.length === 0 ? (
        <EmptyState>Aucun commentaire dans cette catégorie.</EmptyState>
      ) : (
        <ul className="space-y-4">
          {ratings.map((rating) => (
            <li key={rating.incidentId}>
              <CommentCard rating={rating} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}
