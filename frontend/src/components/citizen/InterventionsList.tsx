import { Avatar } from "@/components/ui/Avatar";
import { RatingStars } from "@/components/ui/RatingStars";
import { getMyInterventions } from "@/lib/api";
import { formatDate, formatDateTime } from "@/lib/format";
import { requireCitizenSession } from "@/lib/server-session";
import { RatingForm } from "./RatingForm";

const outcomeLabels = { reussi: "Intervention réussie", echoue: "Intervention échouée" } as const;

/** Super-héros à noter / déjà notés. Lit la session : à rendre sous <Suspense>. */
export async function InterventionsList() {
  const token = await requireCitizenSession();
  const interventions = await getMyInterventions(token);

  if (interventions.length === 0) {
    return (
      <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
        Aucune intervention terminée pour l&apos;instant. Vous pourrez noter les super-héros dès
        qu&apos;une intervention sur un incident que vous avez déclaré sera terminée.
      </p>
    );
  }

  const toRate = interventions.filter(({ review }) => !review).length;

  return (
    <>
      <p className="mb-6 text-sm text-muted" role="status">
        {toRate === 0
          ? "Vous avez noté tous les super-héros intervenus. Merci !"
          : `${toRate} super-héros en attente de votre avis.`}
      </p>
      <ul className="space-y-6">
        {interventions.map(({ interventionId, incident, outcome, endedAt, hero, review }) => {
          const titleId = `intervention-${interventionId}-${hero.id}`;
          return (
            <li key={`${interventionId}-${hero.id}`}>
              <article
                aria-labelledby={titleId}
                className="grid gap-6 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8 md:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]"
              >
                <div>
                  <div className="flex items-center gap-3">
                    <Avatar name={hero.alias} src={null} size={48} />
                    <div>
                      <p className="text-sm text-muted">Super-héros intervenu</p>
                      <p className="font-display text-lg font-semibold">{hero.alias}</p>
                    </div>
                  </div>
                  <h2 id={titleId} className="mt-4 font-display font-semibold">
                    {incident.label ?? `Incident du ${formatDate(endedAt)}`}
                  </h2>
                  <p className="mt-1 text-sm text-muted">
                    {outcomeLabels[outcome]} · Terminée le{" "}
                    <time dateTime={endedAt}>{formatDate(endedAt)}</time>
                  </p>
                </div>

                <div className="md:border-l md:border-border md:pl-6">
                  {review ? (
                    <div>
                      <p className="mb-2 text-sm font-medium">Votre avis</p>
                      <RatingStars rating={review.rating} />
                      {review.comment && (
                        <blockquote className="mt-3 border-l-2 border-border pl-3 text-sm text-pretty">
                          {review.comment}
                        </blockquote>
                      )}
                      <p className="mt-3 text-xs text-muted">
                        Envoyé le{" "}
                        <time dateTime={review.createdAt}>{formatDateTime(review.createdAt)}</time>
                        {review.status === "rejete" && " · Masqué par la modération"}
                      </p>
                    </div>
                  ) : (
                    <RatingForm interventionId={interventionId} hero={hero} />
                  )}
                </div>
              </article>
            </li>
          );
        })}
      </ul>
    </>
  );
}
