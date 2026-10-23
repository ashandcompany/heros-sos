import { Avatar } from "@/components/ui/Avatar";
import { Badge } from "@/components/ui/Badge";
import { DialogTrigger } from "@/components/ui/DialogTrigger";
import { getSuperHeroProfiles } from "@/lib/api";
import { requireAdminSession } from "@/lib/server-session";
import { formatFullName, formatInteger, formatPhone, formatRating, phoneHref } from "@/lib/format";
import type { SuperHeroProfile } from "@/types";
import { heroDialogId, SuperHeroDetailsDialog } from "./SuperHeroDetailsDialog";

const detailsButtonClasses =
  "inline-flex min-h-10 items-center justify-center rounded-full border border-border px-4 text-sm font-semibold whitespace-nowrap hover:border-muted hover:bg-surface-raised";

function FullName({ hero }: { hero: SuperHeroProfile }) {
  const fullName = formatFullName(hero.lastName, hero.firstName);
  return fullName ? <>{fullName}</> : <span className="text-muted italic">Non communiqué</span>;
}

/** Un héros dont le compte est désactivé reste dans l'annuaire, mais ne peut plus se connecter. */
function DeactivatedBadge({ hero }: { hero: SuperHeroProfile }) {
  return hero.accountStatus === "desactive" ? <Badge tone="danger">Compte désactivé</Badge> : null;
}

function DetailsButton({ hero }: { hero: SuperHeroProfile }) {
  return (
    <DialogTrigger dialogId={heroDialogId(hero.id)} className={detailsButtonClasses}>
      Voir la fiche<span className="sr-only"> de {hero.alias}</span>
    </DialogTrigger>
  );
}

function Stats({ heroes }: { heroes: SuperHeroProfile[] }) {
  const totalInterventions = heroes.reduce((sum, hero) => sum + hero.interventionsCount, 0);
  const averageRating =
    heroes.length > 0
      ? heroes.reduce((sum, hero) => sum + hero.averageRating, 0) / heroes.length
      : 0;
  const anonymous = heroes.filter((hero) => !hero.firstName && !hero.lastName).length;

  const stats = [
    { label: "Super-héros inscrits", value: formatInteger(heroes.length) },
    { label: "Interventions au total", value: formatInteger(totalInterventions) },
    { label: "Note moyenne", value: `${formatRating(averageRating)} / 5` },
    { label: "Identité non communiquée", value: formatInteger(anonymous) },
  ];

  return (
    <dl className="mb-8 grid grid-cols-2 gap-3 lg:grid-cols-4">
      {stats.map((stat) => (
        <div
          key={stat.label}
          className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-4"
        >
          <dt className="text-sm text-muted">{stat.label}</dt>
          <dd className="mt-1 font-display text-2xl font-bold">{stat.value}</dd>
        </div>
      ))}
    </dl>
  );
}

/** Liste des super-héros (back-office). Lit la session : à rendre sous <Suspense>. */
export async function SuperHeroesDirectory() {
  const sessionToken = await requireAdminSession();
  const heroes = await getSuperHeroProfiles(sessionToken);

  if (heroes.length === 0) {
    return <p className="text-muted">Aucun super-héros inscrit pour le moment.</p>;
  }

  return (
    <>
      <Stats heroes={heroes} />

      {/* Mobile : une carte par héros */}
      <ul className="space-y-3 md:hidden">
        {heroes.map((hero) => (
          <li
            key={hero.id}
            className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-4"
          >
            <div className="flex items-center gap-3">
              <Avatar name={hero.alias} src={hero.avatarUrl} size={44} />
              <div className="min-w-0">
                <p className="font-display font-semibold">{hero.alias}</p>
                <p className="text-sm">
                  <FullName hero={hero} />
                </p>
                <DeactivatedBadge hero={hero} />
              </div>
            </div>
            <dl className="mt-4 space-y-2 text-sm">
              <div>
                <dt className="sr-only">Adresse email</dt>
                <dd className="break-all">
                  <a href={`mailto:${hero.email}`} className="text-accent hover:underline">
                    {hero.email}
                  </a>
                </dd>
              </div>
              <div>
                <dt className="sr-only">Numéro de portable</dt>
                <dd>
                  <a href={phoneHref(hero.mobilePhone)} className="text-accent hover:underline">
                    {formatPhone(hero.mobilePhone)}
                  </a>
                </dd>
              </div>
            </dl>
            <div className="mt-4">
              <DetailsButton hero={hero} />
            </div>
          </li>
        ))}
      </ul>

      {/* Desktop : tableau */}
      <div className="hidden overflow-x-auto rounded-2xl border border-border md:block">
        <table className="w-full text-left text-sm">
          <caption className="sr-only">
            Super-héros inscrits : pseudo, nom et prénom, email, téléphone
          </caption>
          <thead className="bg-surface-raised text-muted">
            <tr>
              <th scope="col" className="px-4 py-3 font-medium">
                Pseudo
              </th>
              <th scope="col" className="px-4 py-3 font-medium">
                Nom – Prénom
              </th>
              <th scope="col" className="px-4 py-3 font-medium">
                Adresse email
              </th>
              <th scope="col" className="px-4 py-3 font-medium">
                Téléphone
              </th>
              <th scope="col" className="px-4 py-3 font-medium">
                <span className="sr-only">Actions</span>
              </th>
            </tr>
          </thead>
          <tbody className="divide-y divide-border bg-surface">
            {heroes.map((hero) => (
              <tr key={hero.id} className="hover:bg-surface-raised/50">
                <th scope="row" className="px-4 py-3 font-normal">
                  <span className="flex items-center gap-3">
                    <Avatar name={hero.alias} src={hero.avatarUrl} size={36} />
                    <span className="font-display font-semibold">{hero.alias}</span>
                    <DeactivatedBadge hero={hero} />
                  </span>
                </th>
                <td className="px-4 py-3">
                  <FullName hero={hero} />
                </td>
                <td className="px-4 py-3">
                  <a href={`mailto:${hero.email}`} className="text-accent hover:underline">
                    {hero.email}
                  </a>
                </td>
                <td className="px-4 py-3 whitespace-nowrap">
                  <a href={phoneHref(hero.mobilePhone)} className="text-accent hover:underline">
                    {formatPhone(hero.mobilePhone)}
                  </a>
                </td>
                <td className="px-4 py-3 text-right">
                  <DetailsButton hero={hero} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      {heroes.map((hero) => (
        <SuperHeroDetailsDialog key={hero.id} hero={hero} />
      ))}
    </>
  );
}

export function SuperHeroesDirectorySkeleton() {
  return (
    <div aria-hidden="true">
      <div className="mb-8 grid grid-cols-2 gap-3 lg:grid-cols-4">
        {Array.from({ length: 4 }, (_, index) => (
          <div
            key={index}
            className="h-22 border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        ))}
      </div>
      <div className="h-96 border-2 border-border bg-surface motion-safe:animate-pulse" />
    </div>
  );
}
