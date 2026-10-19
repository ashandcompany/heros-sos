import { EmptyState } from "@/components/admin/ModerationFilter";
import { RatingStars } from "@/components/ui/RatingStars";
import { getMyReputation } from "@/lib/api";
import { formatDateTime, formatInteger, formatRating } from "@/lib/format";
import { requireHeroSession } from "@/lib/server-session";
import type { ReceivedRating } from "@/types";
import { ReplyForm } from "./ReplyForm";

function RatingComment({ rating }: { rating: ReceivedRating }) {
  if (rating.comment) {
    return (
      <blockquote className="mt-3 border-l-2 border-border pl-3 text-pretty">
        {rating.comment}
      </blockquote>
    );
  }
  if (rating.commentModeration === "en-attente") {
    return (
      <p className="mt-3 text-sm text-muted">
        Commentaire en cours de vérification par la modération.
      </p>
    );
  }
  if (rating.commentModeration === "rejete") {
    return <p className="mt-3 text-sm text-muted">Commentaire masqué par la modération.</p>;
  }
  return <p className="mt-3 text-sm text-muted">Note sans commentaire.</p>;
}

/** Note globale et avis des citoyens. Lit la session : à rendre sous <Suspense>. */
export async function HeroReputationContent() {
  const { token } = await requireHeroSession();
  const { averageRating, ratingsCount, ratings } = await getMyReputation(token);

  return (
    <div className="grid gap-6 lg:grid-cols-[18rem_minmax(0,1fr)] lg:items-start">
      <section
        aria-labelledby="score-title"
        className="rounded-2xl border border-border bg-surface p-5 sm:p-6"
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
              Sur {formatInteger(ratingsCount)} avis de citoyens
            </p>
          </>
        )}
      </section>

      <section aria-labelledby="reviews-title">
        <h2 id="reviews-title" className="mb-4 font-display text-xl font-semibold">
          Avis des citoyens
        </h2>
        {ratings.length === 0 ? (
          <EmptyState>
            Aucun avis pour l&apos;instant. Les citoyens peuvent vous noter après chaque
            intervention réussie.
          </EmptyState>
        ) : (
          <ul className="space-y-4">
            {ratings.map((rating) => {
              const titleId = `avis-${rating.incidentId}`;
              return (
                <li key={rating.incidentId}>
                  <article
                    aria-labelledby={titleId}
                    className="rounded-2xl border border-border bg-surface p-5 sm:p-6"
                  >
                    <div className="flex flex-wrap items-center justify-between gap-2">
                      <RatingStars rating={rating.rating} />
                      <time dateTime={rating.ratedAt} className="font-mono text-xs text-muted">
                        {formatDateTime(rating.ratedAt)}
                      </time>
                    </div>
                    <h3 id={titleId} className="mt-2 text-sm text-muted">
                      Intervention :{" "}
                      <span className="text-foreground">{rating.incident.title}</span>
                    </h3>
                    <RatingComment rating={rating} />

                    {rating.reply ? (
                      <div className="mt-4 rounded-xl border border-border bg-background p-4">
                        <p className="text-sm font-semibold">Votre réponse</p>
                        <p className="mt-1 text-sm text-pretty">{rating.reply.content}</p>
                        <p className="mt-2 text-xs text-muted">
                          <time dateTime={rating.reply.repliedAt}>
                            {formatDateTime(rating.reply.repliedAt)}
                          </time>
                        </p>
                      </div>
                    ) : (
                      rating.comment && (
                        <div className="mt-4 border-t border-border pt-4">
                          <ReplyForm incidentId={rating.incidentId} />
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
