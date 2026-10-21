import Link from "next/link";
import { moderateReplyAction, moderateReviewAction } from "@/app/actions/admin";
import { cn } from "@/components/ui/cn";
import { RatingStars } from "@/components/ui/RatingStars";
import { getRepliesForModeration, getReviewsForModeration } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import type { HeroReview, ModerationStatus, ReplyForModeration } from "@/types";
import { decisionsFor } from "./decisions";
import { ModerationBadge } from "./ModerationBadge";
import { EmptyState, ModerationFilter, readModerationFilter } from "./ModerationFilter";
import { ModerationForm } from "./ModerationForm";

/*
 * Avis des citoyens (publiés dès leur envoi : la modération masque après coup) et réponses des
 * super-héros (publiées à leur création, à revérifier après chaque modification).
 */

export type CommentsSearchParams = Promise<{
  statut?: string | string[];
  type?: string | string[];
}>;

const repliesPath = `${routes.adminComments}?type=reponses`;

const reviewStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "À modérer",
  valide: "Publié",
  rejete: "Masqué",
};

const replyStatusLabels: Record<ModerationStatus, string> = {
  "en-attente": "À vérifier",
  valide: "Publiée",
  rejete: "Masquée",
};

function ReviewCard({ review }: { review: HeroReview }) {
  const titleId = `avis-${review.id}-titre`;

  return (
    <article
      aria-labelledby={titleId}
      className="grid gap-5 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto]"
    >
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-3">
          <ModerationBadge status={review.status} labels={reviewStatusLabels} />
          <RatingStars rating={review.rating} />
        </div>
        <h2 id={titleId} className="mt-3 font-display font-semibold">
          Avis sur {review.hero.alias}
        </h2>
        <blockquote className="mt-2 border-l-2 border-border pl-3 text-sm text-pretty">
          {review.comment}
        </blockquote>
        <p className="mt-2 text-xs text-muted">
          Envoyé le{" "}
          <time dateTime={review.createdAt} className="font-mono">
            {formatDateTime(review.createdAt)}
          </time>
        </p>
      </div>
      <ModerationForm
        action={moderateReviewAction}
        id={review.id}
        subject={`avis sur ${review.hero.alias}`}
        decisions={decisionsFor(review.status, {
          approve: "Publier",
          reject: "Masquer",
          restore: "Publier",
        })}
      />
    </article>
  );
}

function ReplyCard({ item }: { item: ReplyForModeration }) {
  const titleId = `reponse-${item.reviewId}-titre`;
  const { reply } = item;

  return (
    <article
      aria-labelledby={titleId}
      className="grid gap-5 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto]"
    >
      <div className="min-w-0">
        <ModerationBadge status={reply.status} labels={replyStatusLabels} />
        <h2 id={titleId} className="mt-3 font-display font-semibold">
          Réponse de {item.heroAlias}
        </h2>
        <blockquote className="mt-2 border-l-2 border-primary pl-3 text-sm text-pretty">
          {reply.content}
        </blockquote>
        <p className="mt-2 text-xs text-muted">
          {reply.editedAt ? "Modifiée le " : "Publiée le "}
          <time dateTime={reply.editedAt ?? reply.createdAt} className="font-mono">
            {formatDateTime(reply.editedAt ?? reply.createdAt)}
          </time>
        </p>
        <div className="mt-4 rounded-xl border border-border bg-background p-4">
          <div className="flex flex-wrap items-center gap-3">
            <p className="text-sm font-semibold">En réponse à l&apos;avis</p>
            <RatingStars rating={item.rating} />
            {item.reviewStatus === "rejete" && (
              <ModerationBadge status="rejete" labels={reviewStatusLabels} />
            )}
          </div>
          {item.comment && <p className="mt-2 text-sm text-pretty text-muted">{item.comment}</p>}
        </div>
      </div>
      <ModerationForm
        action={moderateReplyAction}
        id={item.reviewId}
        subject={`réponse de ${item.heroAlias}`}
        decisions={decisionsFor(reply.status, {
          approve: "Publier",
          reject: "Masquer",
          restore: "Publier",
        })}
      />
    </article>
  );
}

function Tabs({ current }: { current: "avis" | "reponses" }) {
  const tabs = [
    { id: "avis", label: "Avis des citoyens", href: routes.adminComments },
    { id: "reponses", label: "Réponses des super-héros", href: repliesPath },
  ] as const;

  return (
    <nav aria-label="Type de commentaire" className="mb-4 border-b-2 border-border">
      <ul className="-mb-0.5 flex flex-wrap gap-4">
        {tabs.map((tab) => (
          <li key={tab.id}>
            <Link
              href={tab.href}
              aria-current={current === tab.id ? "page" : undefined}
              className={cn(
                "inline-flex min-h-11 items-center border-b-2 px-1 font-medium",
                current === tab.id
                  ? "border-primary text-foreground"
                  : "border-transparent text-muted hover:text-foreground",
              )}
            >
              {tab.label}
            </Link>
          </li>
        ))}
      </ul>
    </nav>
  );
}

async function ReviewsList({ searchParams }: { searchParams: CommentsSearchParams }) {
  const token = await requireAdminSession();
  const filter = await readModerationFilter(searchParams, "valide");
  const reviews = await getReviewsForModeration(token, filter === "en-attente" ? "valide" : filter);

  return (
    <>
      <p className="mb-4 text-sm text-muted">
        Les avis sont publiés dès leur envoi. Masquez les propos injurieux ou les données
        personnelles : un avis masqué disparaît de la fiche du super-héros et ne compte plus dans sa
        note.
      </p>
      <ModerationFilter
        basePath={routes.adminComments}
        current={filter === "en-attente" ? "valide" : filter}
        options={["valide", "rejete", "tous"]}
        defaultOption="valide"
        labels={{ valide: "Publiés", rejete: "Masqués", tous: "Tous" }}
      />
      {reviews.length === 0 ? (
        <EmptyState>Aucun avis commenté dans cette catégorie.</EmptyState>
      ) : (
        <ul className="space-y-4">
          {reviews.map((review) => (
            <li key={review.id}>
              <ReviewCard review={review} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

async function RepliesList({ searchParams }: { searchParams: CommentsSearchParams }) {
  const token = await requireAdminSession();
  const filter = await readModerationFilter(searchParams);
  const replies = await getRepliesForModeration(token, filter);

  return (
    <>
      <p className="mb-4 text-sm text-muted">
        Une réponse est publiée à sa création ; chaque modification par le super-héros la remet en
        vérification.
      </p>
      <ModerationFilter
        basePath={repliesPath}
        current={filter}
        labels={{
          "en-attente": "À vérifier",
          valide: "Publiées",
          rejete: "Masquées",
          tous: "Toutes",
        }}
      />
      {replies.length === 0 ? (
        <EmptyState>Aucune réponse dans cette catégorie.</EmptyState>
      ) : (
        <ul className="space-y-4">
          {replies.map((item) => (
            <li key={item.reviewId}>
              <ReplyCard item={item} />
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

/** Avis et réponses à modérer. Lit la session et l'URL : à rendre sous <Suspense>. */
export async function CommentsModeration({ searchParams }: { searchParams: CommentsSearchParams }) {
  const raw = (await searchParams).type;
  const type = (Array.isArray(raw) ? raw[0] : raw) === "reponses" ? "reponses" : "avis";

  return (
    <>
      <Tabs current={type} />
      {type === "reponses" ? (
        <RepliesList searchParams={searchParams} />
      ) : (
        <ReviewsList searchParams={searchParams} />
      )}
    </>
  );
}
