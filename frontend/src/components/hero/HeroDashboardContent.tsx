import Link from "next/link";
import type { ReactNode } from "react";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { RatingStars } from "@/components/ui/RatingStars";
import { getHeroDashboard } from "@/lib/api";
import { formatInteger, incidentTypeLabels } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireHeroSession } from "@/lib/server-session";
import type { IncidentType } from "@/types";
import { HeroIncidentCard } from "./HeroIncidentCard";

function StatTile({
  label,
  value,
  detail,
}: {
  label: string;
  value: ReactNode;
  detail?: ReactNode;
}) {
  return (
    <div className="flex flex-col rounded-2xl border border-border bg-surface p-5">
      <dt className="order-2 text-sm text-muted">{label}</dt>
      <dd className="order-1 font-mono text-4xl font-semibold">{value}</dd>
      {detail && <dd className="order-3 mt-2 text-sm text-muted">{detail}</dd>}
    </div>
  );
}

function SectionTitle({ id, children }: { id: string; children: ReactNode }) {
  return (
    <h2 id={id} className="mb-4 font-display text-xl font-semibold">
      {children}
    </h2>
  );
}

/** Incidents par type : barres horizontales d'une seule teinte, valeurs écrites en clair. */
function IncidentTypesBreakdown({ counts }: { counts: Record<IncidentType, number> }) {
  const rows = (Object.entries(counts) as Array<[IncidentType, number]>).sort(
    ([, a], [, b]) => b - a,
  );
  const max = Math.max(1, ...rows.map(([, count]) => count));

  return (
    <div className="rounded-2xl border border-border bg-surface p-5">
      <h3 id="types-title" className="mb-4 text-sm text-muted">
        Signalements par type d&apos;incident
      </h3>
      <table aria-labelledby="types-title" className="w-full text-sm">
        <thead className="sr-only">
          <tr>
            <th scope="col">Type</th>
            <th scope="col">Signalements</th>
          </tr>
        </thead>
        <tbody>
          {rows.map(([type, count]) => (
            <tr key={type}>
              <th scope="row" className="w-36 py-1.5 pr-3 text-left font-normal">
                {incidentTypeLabels[type]}
              </th>
              <td className="py-1.5">
                <span className="flex items-center gap-3">
                  <span aria-hidden="true" className="h-2.5 flex-1">
                    <span
                      className="block h-full rounded-r-sm bg-accent"
                      style={{ width: `${(count / max) * 100}%` }}
                    />
                  </span>
                  <span className="w-8 text-right font-mono font-semibold">
                    {formatInteger(count)}
                  </span>
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

/** Contenu du tableau de bord. Lit la session : à rendre sous <Suspense>. */
export async function HeroDashboardContent() {
  const { token, user } = await requireHeroSession();
  const { global, personal, myOngoingIncidents, recentIncidents } = await getHeroDashboard(token);

  return (
    <div className="space-y-12">
      <section aria-labelledby="ongoing-title">
        <SectionTitle id="ongoing-title">Mes interventions en cours</SectionTitle>
        {myOngoingIncidents.length === 0 ? (
          <EmptyState>
            Aucune intervention en cours.{" "}
            <Link href={routes.heroIncidents} className="text-accent hover:underline">
              Consulter les signalements
            </Link>
          </EmptyState>
        ) : (
          <ul className="grid gap-3 md:grid-cols-2">
            {myOngoingIncidents.map((incident) => (
              <li key={incident.id}>
                <HeroIncidentCard incident={incident} heroId={user.id} />
              </li>
            ))}
          </ul>
        )}
      </section>

      <section aria-labelledby="personal-title">
        <SectionTitle id="personal-title">Mes statistiques</SectionTitle>
        <dl className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          <StatTile
            label="Interventions"
            value={formatInteger(personal.interventions)}
            detail={`Récemment : ${personal.successfulInterventions} réussie${personal.successfulInterventions > 1 ? "s" : ""}, ${personal.failedInterventions} échouée${personal.failedInterventions > 1 ? "s" : ""}`}
          />
          <StatTile
            label={`Note moyenne (${personal.ratingsCount} avis)`}
            value={
              personal.ratingsCount > 0 ? <RatingStars rating={personal.averageRating} /> : "—"
            }
            detail={
              <Link href={routes.heroReputation} className="text-accent hover:underline">
                Voir ma réputation
              </Link>
            }
          />
          <StatTile
            label="Films dans ma vidéothèque"
            value={formatInteger(personal.videos)}
            detail={`${personal.publishedVideos} publié${personal.publishedVideos > 1 ? "s" : ""}`}
          />
          <div className="flex flex-col rounded-2xl border border-border bg-surface p-5">
            <dt className="text-sm text-muted">Ennemis principaux</dt>
            <dd className="mt-2">
              {personal.mainEnemies.length === 0 ? (
                <p className="text-sm text-muted">Aucun vilain affronté pour l&apos;instant.</p>
              ) : (
                <ol className="space-y-1 text-sm">
                  {personal.mainEnemies.map((enemy) => (
                    <li key={enemy.id} className="flex justify-between gap-2">
                      <span className="font-medium">{enemy.alias}</span>
                      <span className="font-mono text-muted">
                        {enemy.encounters}
                        <span className="sr-only">
                          {" "}
                          affrontement{enemy.encounters > 1 ? "s" : ""}
                        </span>
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </dd>
          </div>
        </dl>
      </section>

      <section aria-labelledby="global-title">
        <SectionTitle id="global-title">Statistiques de la plateforme</SectionTitle>
        <div className="grid gap-3 lg:grid-cols-[minmax(0,1fr)_minmax(0,1fr)]">
          <dl className="grid gap-3 sm:grid-cols-2">
            <StatTile label="Interventions" value={formatInteger(global.interventions)} />
            <StatTile label="Citoyens secourus" value={formatInteger(global.rescuedCitizens)} />
            <StatTile label="Vilains identifiés" value={formatInteger(global.villainsIdentified)} />
            <StatTile
              label="Vilains emprisonnés"
              value={formatInteger(global.villainsImprisoned)}
            />
          </dl>
          <IncidentTypesBreakdown counts={global.incidentsByType} />
        </div>
      </section>

      <section aria-labelledby="recent-title">
        <div className="mb-4 flex flex-wrap items-baseline justify-between gap-2">
          <h2 id="recent-title" className="font-display text-xl font-semibold">
            Derniers signalements
          </h2>
          <Link
            href={routes.heroIncidents}
            className="text-sm font-semibold text-accent hover:underline"
          >
            Tous les incidents<span aria-hidden="true">&nbsp;→</span>
          </Link>
        </div>
        {recentIncidents.length === 0 ? (
          <EmptyState>Aucun incident en attente. La ville est calme.</EmptyState>
        ) : (
          <ul className="grid gap-3 md:grid-cols-2">
            {recentIncidents.map((incident) => (
              <li key={incident.id}>
                <HeroIncidentCard incident={incident} heroId={user.id} />
              </li>
            ))}
          </ul>
        )}
      </section>
    </div>
  );
}

export function HeroDashboardSkeleton() {
  return (
    <div aria-hidden="true" className="space-y-6">
      <div className="h-40 rounded-2xl border border-border bg-surface motion-safe:animate-pulse" />
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <div
            key={index}
            className="h-36 rounded-2xl border border-border bg-surface motion-safe:animate-pulse"
          />
        ))}
      </div>
    </div>
  );
}
