import Link from "next/link";
import { getModerationSummary } from "@/lib/api";
import { formatInteger } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

interface Tile {
  label: string;
  /** `null` : compteur non disponible (tuile masquée). */
  count: number | null;
  href: string;
  /** Texte du lien, complété par le libellé pour les lecteurs d'écran. */
  cta: string;
}

/** Compteurs des éléments à traiter. Lit la session : à rendre sous <Suspense>. */
export async function DashboardSummary() {
  const token = await requireAdminSession();
  const summary = await getModerationSummary(token);

  const tiles: Tile[] = [
    {
      label: "Incidents à vérifier",
      count: summary.incidents,
      href: routes.adminIncidents,
      cta: "Vérifier",
    },
    {
      label: "Demandes de super-héros",
      count: summary.heroApplications,
      href: routes.adminHeroes,
      cta: "Examiner",
    },
    {
      label: "Nouveaux comptes à contrôler",
      count: summary.accounts,
      href: routes.adminAccounts,
      cta: "Contrôler",
    },
    {
      label: "Changements d'email",
      count: summary.emailChanges,
      href: `${routes.adminAccounts}#emails`,
      cta: "Traiter",
    },
    {
      label: "Films à valider",
      count: summary.videos,
      href: routes.adminVideos,
      cta: "Visionner",
    },
    {
      label: "Réponses de héros à vérifier",
      count: summary.comments,
      href: `${routes.adminComments}?type=reponses`,
      cta: "Modérer",
    },
    {
      label: "Articles en brouillon",
      count: summary.draftArticles,
      href: routes.adminBlog,
      cta: "Ouvrir le blog",
    },
    {
      label: "Newsletters en brouillon",
      count: summary.draftNewsletters,
      href: routes.adminNewsletters,
      cta: "Ouvrir la newsletter",
    },
  ];
  const shown = tiles.filter((tile): tile is Tile & { count: number } => tile.count !== null);
  // Les brouillons (articles, newsletters) ne sont pas des éléments à modérer.
  const drafts: string[] = [routes.adminBlog, routes.adminNewsletters];
  const total = shown
    .filter((tile) => !drafts.includes(tile.href))
    .reduce((sum, tile) => sum + tile.count, 0);

  return (
    <>
      <p role="status" className="mb-6 text-muted">
        {total === 0
          ? "Rien à traiter pour le moment. Tout est à jour !"
          : `${formatInteger(total)} élément${total > 1 ? "s" : ""} en attente de modération.`}
      </p>
      <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
        {shown.map((tile) => (
          <li
            key={tile.label}
            className="flex flex-col border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5"
          >
            <p className="text-sm text-muted">{tile.label}</p>
            <p
              className={
                tile.count > 0
                  ? "mt-1 font-mono text-4xl font-semibold text-accent"
                  : "mt-1 font-mono text-4xl font-semibold text-muted"
              }
            >
              {formatInteger(tile.count)}
            </p>
            <Link
              href={tile.href}
              className="mt-4 inline-flex min-h-11 items-center self-start text-sm font-semibold text-accent hover:underline"
            >
              {tile.cta}
              <span className="sr-only"> : {tile.label.toLowerCase()}</span>
              <span aria-hidden="true">&nbsp;→</span>
            </Link>
          </li>
        ))}
      </ul>
    </>
  );
}

export function DashboardSummarySkeleton() {
  return (
    <div aria-hidden="true" className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
      {Array.from({ length: 8 }, (_, index) => (
        <div
          key={index}
          className="h-40 border-2 border-border bg-surface motion-safe:animate-pulse"
        />
      ))}
    </div>
  );
}
