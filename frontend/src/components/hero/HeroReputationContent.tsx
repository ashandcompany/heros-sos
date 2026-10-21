import { ModerationBadge } from "@/components/admin/ModerationBadge";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { RatingStars } from "@/components/ui/RatingStars";
import { getMyReputation } from "@/lib/api";
import { formatDateTime, formatInteger, formatRating } from "@/lib/format";
import { requireHeroSession } from "@/lib/server-session";
import type { HeroReview, ModerationStatus } from "@/types";
import { ReplyForm } from "./ReplyForm";

const replyStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "En cours de vérification",
  valide: "Publiée",
  rejete: "Masquée par la modération",
};

function ReviewComment({ review }: { review: HeroReview }) {
  if (review.status === "rejete") {
    return (
      <p className="mt-3 text-sm text-muted">
        Avis masqué par la modération : il ne compte pas dans votre note.
      </p>
    );
  }
  if (review.comment) {
    return (
      <blockquote className="mt-3 border-l-2 border-border pl-3 text-pretty">
        {review.comment}
      </blockquote>
    );
  }
  return <p className="mt-3 text-sm text-muted">Note sans commentaire.</p>;
}

/** Note globale et avis des citoyens. Lit la session : à rendre sous <Suspense>. */
export async function HeroReputationContent() {
  const { token } = await requireHeroSession();
  const { averageRating, ratingsCount, reviews } = await getMyReputation(token);

  return (
    <div className="grid gap-6 lg:grid-cols-[18rem_minmax(0,1fr)] lg:items-start">
      <section
        aria-labelledby="score-title"
        className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6"
      >
        <h2 id="score-title" className="text-sm text-muted">
          Note globale
        </h2>
        {ratingsCount === 0 ? (
          <p className="mt-2 text-sm text-muted">Pas encore de note.</p>
        ) : (
          <>
            <p className="mt-1 font-mono text-5xl font-semibold">
              {formatRating(averageRating)}
              <span className="text-2xl text-muted">/5</span>
            </p>
            <div className="mt-2">
              <RatingStars rating={averageRating} />
            </div>
            <p className="mt-2 text-sm text-muted">
              Sur {formatInteger(ratingsCount)} avis publiés
            </p>
          </>
        )}
      </section>

      <section aria-labelledby="reviews-title">
        <h2 id="reviews-title" className="mb-4 font-display text-xl font-semibold">
          Avis des citoyens
        </h2>
        {reviews.length === 0 ? (
          <EmptyState>
            Aucun avis pour l&apos;instant. Les citoyens peuvent vous noter après chaque
            intervention terminée.
          </EmptyState>
        ) : (
          <ul className="space-y-4">
            {reviews.map((review) => {
              const titleId = `avis-${review.id}`;
              return (
                <li key={review.id}>
                  <article
                    aria-labelledby={titleId}
                    className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <RatingStars rating={review.rating} />
                      <h3 id={titleId} className="font-mono text-xs text-muted">
                        <span className="sr-only">Avis reçu le </span>
                        <time dateTime={review.createdAt}>{formatDateTime(review.createdAt)}</time>
                      </h3>
                    </div>
                    <ReviewComment review={review} />

                    {review.reply ? (
                      <div className="mt-4 rounded-xl border border-border bg-background p-4">
                        <div className="flex flex-wrap items-center gap-2">
                          <p className="text-sm font-semibold">Votre réponse</p>
                          <ModerationBadge
                            status={review.reply.status}
                            labels={replyStatusLabels}
                          />
                        </div>
                        <p className="mt-1 text-sm text-pretty">{review.reply.content}</p>
                        <p className="mt-2 text-xs text-muted">
                          <time dateTime={review.reply.editedAt ?? review.reply.createdAt}>
                            {formatDateTime(review.reply.editedAt ?? review.reply.createdAt)}
                          </time>
                          {review.reply.editedAt && " (modifiée)"}
                        </p>
                        {/* L'API n'accepte une réponse que sur un avis publié et commenté. */}
                        {review.status === "valide" && review.comment && (
                          <details className="group mt-3 border-t border-border pt-3">
                            <summary className="inline-flex min-h-11 cursor-pointer items-center text-sm font-semibold text-accent hover:underline">
                              <span className="group-open:hidden">Modifier ma réponse</span>
                              <span className="hidden group-open:inline">
                                Annuler la modification
                              </span>
                            </summary>
                            <div className="mt-2">
                              <ReplyForm reviewId={review.id} currentReply={review.reply.content} />
                            </div>
                          </details>
                        )}
                      </div>
                    ) : (
                      review.status === "valide" &&
                      review.comment && (
                        <div className="mt-4 border-t border-border pt-4">
                          <ReplyForm reviewId={review.id} />
                        </div>
                      )
                    )}
                  </article>
                </li>
              );
            })}
          </ul>
        )}
      </section>
    </div>
  );
}
