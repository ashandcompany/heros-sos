import { Avatar } from "@/components/ui/Avatar";
import { RatingStars } from "@/components/ui/RatingStars";
import { getMyInterventions } from "@/lib/api";
import { formatDate, formatDateTime, incidentTypeLabels } from "@/lib/format";
import { requireCitizenSession } from "@/lib/server-session";
import { RatingForm } from "./RatingForm";

/** Interventions à noter / déjà notées. Lit la session : à rendre sous <Suspense>. */
export async function InterventionsList() {
  const token = await requireCitizenSession();
  const interventions = await getMyInterventions(token);

  if (interventions.length === 0) {
    return (
      <p className="rounded-2xl border border-dashed border-border p-5 text-sm text-muted">
        Aucune intervention terminée pour l&apos;instant. Vous pourrez noter le super-héros dès
        qu&apos;un incident que vous avez déclaré sera résolu.
      </p>
    );
  }

  const toRate = interventions.filter(({ rating }) => !rating).length;

  return (
    <>
      <p className="mb-6 text-sm text-muted" role="status">
        {toRate === 0
          ? "Vous avez noté toutes vos interventions. Merci !"
          : `${toRate} intervention${toRate > 1 ? "s" : ""} en attente de votre avis.`}
      </p>
      <ul className="space-y-6">
        {interventions.map(({ incident, rating }) => {
          const titleId = `intervention-${incident.id}`;
          return (
            <li key={incident.id}>
              <article
                aria-labelledby={titleId}
                className="grid gap-6 rounded-2xl border border-border bg-surface p-5 sm:p-8 md:grid-cols-[minmax(0,1fr)_minmax(0,1.2fr)]"
              >
                <div>
                  <div className="flex items-center gap-3">
                    <Avatar name={incident.hero.alias} src={null} size={48} />
                    <div>
                      <p className="text-sm text-muted">Super-héros intervenu</p>
                      <p className="font-display text-lg font-semibold">{incident.hero.alias}</p>
                    </div>
                  </div>
                  <h2 id={titleId} className="mt-4 font-display font-semibold">
                    {incident.title}
                  </h2>
                  <p className="mt-1 text-sm text-muted">
                    {incidentTypeLabels[incident.type]} · {incident.location.city} · Résolu le{" "}
                    <time dateTime={incident.resolvedAt}>{formatDate(incident.resolvedAt)}</time>
                  </p>
                </div>

                <div className="md:border-l md:border-border md:pl-6">
                  {rating ? (
                    <div>
                      <p className="mb-2 text-sm font-medium">Votre avis</p>
                      <RatingStars rating={rating.rating} />
                      {rating.comment && (
                        <blockquote className="mt-3 border-l-2 border-border pl-3 text-sm text-pretty">
                          {rating.comment}
                        </blockquote>
                      )}
                      <p className="mt-3 text-xs text-muted">
                        Envoyé le{" "}
                        <time dateTime={rating.ratedAt}>{formatDateTime(rating.ratedAt)}</time>
                      </p>
                    </div>
                  ) : (
                    <RatingForm incidentId={incident.id} heroAlias={incident.hero.alias} />
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
