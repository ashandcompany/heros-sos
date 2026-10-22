import { notFound } from "next/navigation";
import { Badge } from "@/components/ui/Badge";
import { canSetPriority, getHeroIncident, getSuperHeroes } from "@/lib/api";
import { formatDateTime, interventionOutcomeLabels, reporterRoleLabels } from "@/lib/format";
import { requireHeroSession } from "@/lib/server-session";
import {
  FinishInterventionForm,
  InvitationResponseForm,
  InviteHeroesForm,
  JoinTeamForm,
  OutcomeVoteForm,
  PriorityForm,
  TakeChargeForm,
} from "./IncidentActions";
import { PriorityBadge, StatusBadge } from "./IncidentBadges";
import { TeamMembersList } from "./TeamMembersList";

const panelClasses =
  "border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-6";

/** Lit la session et les paramètres d'URL : à rendre sous <Suspense>. */
export async function HeroIncidentDetail({ params }: { params: Promise<{ id: string }> }) {
  const { token, user } = await requireHeroSession();
  const { id } = await params;
  const incident = await getHeroIncident(token, id);
  if (!incident) notFound();

  const place = [incident.location.address, incident.location.district, incident.location.city]
    .filter(Boolean)
    .join(", ");
  const { team } = incident;
  const me = team?.members.find(({ isMe }) => isMe);
  const interventionRef = team && { incidentId: incident.id, interventionId: team.id };
  const solo =
    (team?.members.filter(({ participation }) => participation === "confirme") ?? []).length <= 1;
  // Renforts possibles : les super-héros publics qui ne sont pas déjà dans l'équipe.
  const candidates =
    team && me?.participation === "confirme" && !team.proposedOutcome
      ? (await getSuperHeroes())
          .filter(({ id: heroId }) => !team.members.some(({ hero }) => hero.id === heroId))
          .map(({ id: heroId, alias }) => ({ id: heroId, alias }))
      : [];
  const pastInterventions = incident.interventions;

  return (
    <>
      <div className="mb-8">
        <div className="flex flex-wrap items-center gap-2">
          <PriorityBadge priority={incident.priority} />
          <StatusBadge status={incident.status} />
          <Badge>{incident.type.label}</Badge>
        </div>
        <h1 className="mt-3 font-display text-3xl font-bold">{incident.title}</h1>
        <p className="mt-2 text-sm text-muted">
          Signalé le{" "}
          <time dateTime={incident.reportedAt} className="font-mono">
            {formatDateTime(incident.reportedAt)}
          </time>
          {incident.reporterRole &&
            ` par un ${reporterRoleLabels[incident.reporterRole].toLowerCase()}`}
        </p>
      </div>

      <div className="grid gap-6 lg:grid-cols-[minmax(0,1fr)_20rem] lg:items-start">
        <div className="space-y-6">
          <section aria-labelledby="report-title" className={panelClasses}>
            <h2 id="report-title" className="mb-4 font-display text-xl font-semibold">
              Signalement
            </h2>
            {incident.description ? (
              <p className="text-pretty">{incident.description}</p>
            ) : (
              <p className="text-sm text-muted">Aucune description fournie.</p>
            )}
            <dl className="mt-6 grid gap-4 sm:grid-cols-2">
              <div>
                <dt className="text-sm text-muted">Lieu</dt>
                <dd>{place}</dd>
              </div>
              <div>
                <dt className="text-sm text-muted">Super-vilain</dt>
                <dd>{incident.villain?.alias ?? "Non identifié"}</dd>
              </div>
              {incident.occurredAt && (
                <div>
                  <dt className="text-sm text-muted">Date des faits</dt>
                  <dd>
                    <time dateTime={incident.occurredAt}>
                      {formatDateTime(incident.occurredAt)}
                    </time>
                  </dd>
                </div>
              )}
              {incident.resolvedAt && (
                <div>
                  <dt className="text-sm text-muted">Résolu le</dt>
                  <dd>
                    <time dateTime={incident.resolvedAt}>
                      {formatDateTime(incident.resolvedAt)}
                    </time>
                  </dd>
                </div>
              )}
            </dl>
          </section>

          {pastInterventions.length > 0 && (
            <section aria-labelledby="history-title" className={panelClasses}>
              <h2 id="history-title" className="mb-4 font-display text-xl font-semibold">
                Interventions précédentes
              </h2>
              <ol className="space-y-3">
                {pastInterventions.map((intervention) => (
                  <li
                    key={intervention.id}
                    className="flex flex-wrap items-center justify-between gap-2 text-sm"
                  >
                    <span>
                      <span className="font-semibold">
                        {intervention.hero.id === user.id ? "Vous" : intervention.hero.alias}
                      </span>{" "}
                      ·{" "}
                      <time dateTime={intervention.startedAt} className="font-mono text-muted">
                        {formatDateTime(intervention.startedAt)}
                      </time>
                    </span>
                    {intervention.outcome && (
                      <Badge tone={intervention.outcome === "reussi" ? "success" : "danger"}>
                        {interventionOutcomeLabels[intervention.outcome]}
                      </Badge>
                    )}
                  </li>
                ))}
              </ol>
            </section>
          )}
        </div>

        <div className="space-y-6">
          <section aria-labelledby="intervention-title" className={panelClasses}>
            <h2 id="intervention-title" className="font-display text-xl font-semibold">
              Intervention
            </h2>
            <div className="mt-4 space-y-4">
              {!team && incident.status === "signale" && (
                <>
                  <p className="text-sm text-muted">
                    Aucun super-héros n&apos;intervient. En prenant l&apos;incident en charge, vous
                    dirigez l&apos;intervention : d&apos;autres héros pourront rejoindre votre
                    équipe.
                  </p>
                  <TakeChargeForm incidentId={incident.id} />
                </>
              )}
              {!team && incident.status === "resolu" && (
                <p className="text-sm">
                  Incident résolu
                  {incident.hero && (
                    <>
                      {" "}
                      par <span className="font-semibold">{incident.hero.alias}</span>
                    </>
                  )}
                  .
                </p>
              )}
              {team && interventionRef && (
                <>
                  <TeamMembersList team={team} />
                  {!me && (
                    <>
                      <p className="text-sm text-muted">
                        Une équipe intervient déjà. Vous pouvez la rejoindre en renfort.
                      </p>
                      <JoinTeamForm interventionRef={interventionRef} />
                    </>
                  )}
                  {me?.participation === "invite" && (
                    <>
                      <p className="text-sm">
                        <span className="font-semibold text-accent">Vous êtes invité</span> à
                        rejoindre l&apos;équipe. Confirmez votre présence.
                      </p>
                      <InvitationResponseForm interventionRef={interventionRef} />
                    </>
                  )}
                  {me?.participation === "confirme" && !team.proposedOutcome && (
                    <>
                      <p className="text-sm">
                        <span className="font-semibold text-accent">Vous intervenez</span> sur cet
                        incident.{" "}
                        {solo
                          ? "Indiquez le résultat une fois l'intervention terminée."
                          : "Une fois l'intervention terminée, proposez le résultat : toute l'équipe devra le valider."}
                      </p>
                      <FinishInterventionForm interventionRef={interventionRef} solo={solo} />
                    </>
                  )}
                  {me?.participation === "confirme" &&
                    team.proposedOutcome &&
                    (me.vote === "en-attente" ? (
                      <>
                        <p className="text-sm text-muted">
                          L&apos;intervention sera close quand toute l&apos;équipe aura validé ce
                          résultat. Si vous le contestez, la modération tranchera.
                        </p>
                        <OutcomeVoteForm interventionRef={interventionRef} />
                      </>
                    ) : (
                      <p className="text-sm text-muted">
                        Vous avez validé le résultat. En attente du vote des autres membres.
                      </p>
                    ))}
                </>
              )}
            </div>
          </section>

          {team && interventionRef && me?.participation === "confirme" && !team.proposedOutcome && (
            <section aria-labelledby="invite-title" className={panelClasses}>
              <h2 id="invite-title" className="mb-1 font-display text-xl font-semibold">
                Renforts
              </h2>
              <p className="mb-4 text-sm text-muted">
                Chaque super-héros invité reçoit un mail et doit confirmer sa présence.
              </p>
              <InviteHeroesForm interventionRef={interventionRef} candidates={candidates} />
            </section>
          )}

          {canSetPriority(incident) && (
            <section aria-labelledby="priority-title" className={panelClasses}>
              <h2 id="priority-title" className="mb-4 font-display text-xl font-semibold">
                Priorité
              </h2>
              <PriorityForm incidentId={incident.id} priority={incident.priority} />
            </section>
          )}
        </div>
      </div>
    </>
  );
}
