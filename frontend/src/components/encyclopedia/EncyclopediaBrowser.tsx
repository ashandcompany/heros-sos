import Form from "next/form";
import Link from "next/link";
import type { ReactNode } from "react";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { Pagination } from "@/components/admin/Pagination";
import { SuperHeroCard } from "@/components/home/SuperHeroCard";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { SelectField } from "@/components/ui/SelectField";
import { getSuperHeroes, getVillains } from "@/lib/api";
import {
  type ActiveFilter,
  type EncyclopediaQuery,
  type EncyclopediaTab,
  activeFilters,
  encyclopediaParams as params,
  filterHeroes,
  filterQuery,
  filterVillains,
  heroSorts,
  minRatingOptions,
  paginate,
  readEncyclopediaQuery,
  threatLevels,
  villainSorts,
  villainSpecialties,
  villainStatuses,
} from "@/lib/encyclopedia";
import { formatInteger, threatLevelLabels, villainStatusLabels } from "@/lib/format";
import { encyclopediaVillainsRoute, routes } from "@/lib/routes";
import type { Page, SuperHero, Villain } from "@/types";
import { AutoSubmit } from "./AutoSubmit";
import { VillainCard } from "./VillainCard";

/** Multiple de 2, 3 et 4 : les pages pleines remplissent la grille à toutes les largeurs. */
const PAGE_SIZE = 12;
/** Ancre des résultats : la pagination y ramène au lieu du haut de page. */
const RESULTS_ANCHOR = "resultats";

const plural = (count: number, singular: string, pluralForm = `${singular}s`) =>
  `${formatInteger(count)} ${count > 1 ? pluralForm : singular}`;

/**
 * Encyclopédie : onglets héros / vilains, recherche, filtres, tri et pagination, tous lus dans
 * l'URL. Rendu sous <Suspense>.
 */
export async function EncyclopediaBrowser({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const query = readEncyclopediaQuery(await searchParams);
  const [heroes, villains] = await Promise.all([getSuperHeroes(), getVillains()]);

  return (
    <>
      <EncyclopediaTabs
        current={query.tab}
        heroesCount={heroes.length}
        villainsCount={villains.length}
      />
      {query.tab === "heros" ? (
        <HeroesPanel heroes={heroes} query={query} />
      ) : (
        <VillainsPanel villains={villains} query={query} />
      )}
    </>
  );
}

function EncyclopediaTabs({
  current,
  heroesCount,
  villainsCount,
}: {
  current: EncyclopediaTab;
  heroesCount: number;
  villainsCount: number;
}) {
  const tabs = [
    { tab: "heros", label: "Super-héros", count: heroesCount, href: routes.encyclopedia },
    {
      tab: "vilains",
      label: "Super-vilains",
      count: villainsCount,
      href: encyclopediaVillainsRoute,
    },
  ] as const;

  return (
    <nav aria-label="Sections de l'encyclopédie" className="mt-8 border-b-2 border-border">
      <ul className="-mb-0.5 flex flex-wrap gap-2">
        {tabs.map(({ tab, label, count, href }) => {
          const isCurrent = tab === current;
          return (
            <li key={tab}>
              <Link
                href={href}
                aria-current={isCurrent ? "page" : undefined}
                className={cn(
                  "inline-flex min-h-12 items-center gap-3 border-2 px-5 font-display font-bold tracking-wider uppercase",
                  isCurrent
                    ? "border-border border-b-background bg-background text-foreground"
                    : "border-transparent text-muted hover:text-foreground",
                )}
              >
                {label}
                <span
                  className={cn(
                    "border-2 px-1.5 font-mono text-xs",
                    isCurrent ? "border-primary bg-primary/15" : "border-border",
                  )}
                >
                  {formatInteger(count)}
                </span>
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

/* ---------- Super-héros ---------- */

function HeroesPanel({ heroes, query }: { heroes: SuperHero[]; query: EncyclopediaQuery }) {
  const filters = query.heroes;
  const matching = filterHeroes(heroes, filters);
  const page = paginate(matching, query.page, PAGE_SIZE);

  return (
    <section aria-labelledby="results-title" className="mt-6">
      <h2 id="results-title" className="sr-only">
        Super-héros
      </h2>
      <FiltersForm tab="heros" search={filters.search} searchPlaceholder="Pseudo, pouvoir…">
        <SelectField
          label="Note minimale"
          name={params.minRating}
          defaultValue={filters.minRating ?? ""}
          className="sm:w-44"
        >
          <option value="">Toutes</option>
          {Object.entries(minRatingOptions).map(([value, label]) => (
            <option key={value} value={value}>
              {label}
            </option>
          ))}
        </SelectField>
        <SortSelect value={filters.sort} options={heroSorts} />
      </FiltersForm>

      <ResultsSummary
        query={query}
        shown={matching.length}
        total={heroes.length}
        noun={["super-héros", "super-héros"]}
      />

      {heroes.length === 0 ? (
        <EmptyState>Aucun super-héros inscrit pour l&apos;instant.</EmptyState>
      ) : (
        <Results query={query} page={page} emptyMessage="Aucun super-héros ne correspond.">
          <ul className="grid grid-cols-2 gap-4 sm:gap-6 md:grid-cols-3 lg:grid-cols-4">
            {page.items.map((hero) => (
              <li key={hero.id}>
                <SuperHeroCard hero={hero} />
              </li>
            ))}
          </ul>
        </Results>
      )}
    </section>
  );
}

/* ---------- Super-vilains ---------- */

function VillainsPanel({ villains, query }: { villains: Villain[]; query: EncyclopediaQuery }) {
  const filters = query.villains;
  const specialties = villainSpecialties(villains);
  const matching = filterVillains(villains, filters);
  const page = paginate(matching, query.page, PAGE_SIZE);

  return (
    <section aria-labelledby="results-title" className="mt-6">
      <h2 id="results-title" className="sr-only">
        Super-vilains
      </h2>
      <FiltersForm
        tab="vilains"
        search={filters.search}
        searchPlaceholder="Alias, nom, secteur, mode opératoire…"
      >
        <SelectField
          label="Menace"
          name={params.threat}
          defaultValue={filters.threat ?? ""}
          className="sm:w-40"
        >
          <option value="">Toutes</option>
          {threatLevels.map((level) => (
            <option key={level} value={level}>
              {threatLevelLabels[level]}
            </option>
          ))}
        </SelectField>
        <SelectField
          label="Statut"
          name={params.status}
          defaultValue={filters.status ?? ""}
          className="sm:w-40"
        >
          <option value="">Tous</option>
          {villainStatuses.map((status) => (
            <option key={status} value={status}>
              {villainStatusLabels[status]}
            </option>
          ))}
        </SelectField>
        {specialties.length > 0 && (
          <SelectField
            label="Spécialité"
            name={params.specialty}
            defaultValue={filters.specialty ?? ""}
            className="sm:w-52"
          >
            <option value="">Toutes</option>
            {specialties.map(({ code, label }) => (
              <option key={code} value={code}>
                {label}
              </option>
            ))}
          </SelectField>
        )}
        <SortSelect value={filters.sort} options={villainSorts} />
      </FiltersForm>

      <ResultsSummary
        query={query}
        shown={matching.length}
        total={villains.length}
        noun={["super-vilain", "super-vilains"]}
        filters={activeFilters(query, routes.encyclopedia, specialties)}
      />

      {villains.length === 0 ? (
        <EmptyState>Aucun super-vilain répertorié pour l&apos;instant.</EmptyState>
      ) : (
        <Results query={query} page={page} emptyMessage="Aucun super-vilain ne correspond.">
          <ul className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
            {page.items.map((villain) => (
              <li key={villain.id}>
                <VillainCard villain={villain} />
              </li>
            ))}
          </ul>
        </Results>
      )}
    </section>
  );
}

/* ---------- Éléments communs ---------- */

/**
 * Formulaire GET : sans JavaScript, il recharge la page filtrée ; avec, `AutoSubmit` l'applique à
 * chaque changement et `next/form` navigue côté client sans remonter en haut de page.
 * Il ne porte pas le numéro de page : changer un filtre revient à la première.
 */
function FiltersForm({
  tab,
  search,
  searchPlaceholder,
  children,
}: {
  tab: EncyclopediaTab;
  search: string;
  searchPlaceholder: string;
  children: ReactNode;
}) {
  return (
    <Form
      action={routes.encyclopedia}
      replace
      scroll={false}
      role="search"
      aria-label="Filtrer l'encyclopédie"
      className="border-2 border-border bg-surface p-4 shadow-[3px_3px_0_0_var(--color-shadow)] sm:p-5"
    >
      {tab === "vilains" && <input type="hidden" name={params.tab} value="vilains" />}
      <div className="flex flex-col gap-4 sm:flex-row sm:flex-wrap sm:items-end">
        <div className="min-w-0 flex-1 sm:min-w-64">
          <label htmlFor="encyclopedia-search" className="mb-2 block text-sm font-medium">
            Rechercher
          </label>
          <div className="flex">
            <input
              id="encyclopedia-search"
              type="search"
              name={params.search}
              defaultValue={search}
              placeholder={searchPlaceholder}
              autoComplete="off"
              className="min-h-12 w-full min-w-0 border-2 border-r-0 border-border bg-background px-4 text-foreground placeholder:text-muted"
            />
            <button
              type="submit"
              className={cn(buttonClasses("secondary"), "min-h-12 shadow-none hover:shadow-none")}
            >
              <span aria-hidden="true">⌕</span>
              <span className="sr-only sm:not-sr-only">Rechercher</span>
            </button>
          </div>
        </div>
        {children}
      </div>
      <AutoSubmit />
    </Form>
  );
}

function SortSelect<Sort extends string>({
  value,
  options,
}: {
  value: Sort;
  options: Record<Sort, string>;
}) {
  // Le tri par défaut (alias) a la valeur vide : il n'apparaît pas dans l'URL.
  const entries = Object.entries(options) as [Sort, string][];
  return (
    <SelectField
      label="Trier par"
      name={params.sort}
      defaultValue={value === "alias" ? "" : value}
      className="sm:w-56"
    >
      {entries.map(([sort, label]) => (
        <option key={sort} value={sort === "alias" ? "" : sort}>
          {label}
        </option>
      ))}
    </SelectField>
  );
}

/** Nombre de résultats (annoncé aux lecteurs d'écran) et filtres actifs, retirables un à un. */
function ResultsSummary({
  query,
  shown,
  total,
  noun: [singular, pluralForm],
  filters = activeFilters(query, routes.encyclopedia),
}: {
  query: EncyclopediaQuery;
  shown: number;
  total: number;
  noun: [string, string];
  filters?: ActiveFilter[];
}) {
  const resetHref = query.tab === "vilains" ? encyclopediaVillainsRoute : routes.encyclopedia;

  return (
    <div className="mt-6 flex flex-wrap items-center gap-x-4 gap-y-3">
      <p role="status" className="font-mono text-sm tracking-wider text-muted uppercase">
        {shown === total
          ? plural(total, singular, pluralForm)
          : `${plural(shown, singular, pluralForm)} sur ${formatInteger(total)}`}
      </p>
      {filters.length > 0 && (
        <ul aria-label="Filtres actifs" className="flex flex-wrap items-center gap-2">
          {filters.map(({ label, href }) => (
            <li key={label}>
              <Link
                href={href}
                scroll={false}
                replace
                className="inline-flex min-h-9 items-center gap-2 border-2 border-primary bg-primary/15 px-3 text-sm font-medium hover:bg-primary/25"
              >
                {label}
                <span aria-hidden="true">×</span>
                <span className="sr-only">(retirer ce filtre)</span>
              </Link>
            </li>
          ))}
          {filters.length > 1 && (
            <li>
              <Link
                href={resetHref}
                scroll={false}
                replace
                className="text-sm text-accent hover:underline"
              >
                Tout effacer
              </Link>
            </li>
          )}
        </ul>
      )}
    </div>
  );
}

/** Grille de résultats et pagination, ou message si la page est vide. */
function Results({
  query,
  page,
  emptyMessage,
  children,
}: {
  query: EncyclopediaQuery;
  page: Page<unknown>;
  emptyMessage: string;
  children: ReactNode;
}) {
  const filterParams = filterQuery(query);
  const firstPage = new URLSearchParams(
    Object.entries(filterParams).filter((entry): entry is [string, string] => Boolean(entry[1])),
  ).toString();
  const firstPageHref = firstPage ? `${routes.encyclopedia}?${firstPage}` : routes.encyclopedia;
  const resetHref = query.tab === "vilains" ? encyclopediaVillainsRoute : routes.encyclopedia;

  return (
    <div id={RESULTS_ANCHOR} className="mt-4 scroll-mt-24">
      {page.total === 0 ? (
        <EmptyState>
          {emptyMessage} Essayez une autre recherche ou{" "}
          <Link href={resetHref} scroll={false} className="text-accent hover:underline">
            effacez les filtres
          </Link>
          .
        </EmptyState>
      ) : page.items.length === 0 ? (
        // Page au-delà de la dernière (lien ancien ou saisi à la main).
        <EmptyState>
          Cette page n&apos;existe pas.{" "}
          <Link href={firstPageHref} className="text-accent hover:underline">
            Revenir à la première page
          </Link>
        </EmptyState>
      ) : (
        <>
          {children}
          <Pagination
            page={page}
            basePath={routes.encyclopedia}
            query={filterParams}
            param={params.page}
            anchor={RESULTS_ANCHOR}
          />
        </>
      )}
    </div>
  );
}
