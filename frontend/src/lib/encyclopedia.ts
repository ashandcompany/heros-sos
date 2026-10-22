import type {
  Page,
  SuperHero,
  ThreatLevel,
  Villain,
  VillainSpecialty,
  VillainStatus,
} from "@/types";
import { threatLevelLabels, villainStatusLabels } from "./format";

/*
 * Recherche, filtres, tri et pagination de l'encyclopédie, lus dans l'URL.
 * L'API ne filtre les vilains que par début d'alias et ne trie les héros que par pseudo ou note :
 * les listes, courtes, sont chargées en entier et filtrées ici, côté serveur. L'URL reste la seule
 * source de vérité (liens partageables, retour arrière, fonctionne sans JavaScript).
 */

type RawSearchParams = Record<string, string | string[] | undefined>;

/** Paramètres d'URL de l'encyclopédie (en français, comme les routes). */
export const encyclopediaParams = {
  tab: "onglet",
  search: "q",
  sort: "tri",
  minRating: "note",
  threat: "menace",
  status: "statut",
  specialty: "specialite",
  page: "page",
} as const;

export type EncyclopediaTab = "heros" | "vilains";

export const heroSorts = {
  alias: "Nom (A → Z)",
  note: "Mieux notés",
  interventions: "Plus d'interventions",
} as const;
export type HeroSort = keyof typeof heroSorts;

export const villainSorts = {
  alias: "Nom (A → Z)",
  menace: "Plus dangereux",
  incidents: "Plus d'incidents signalés",
} as const;
export type VillainSort = keyof typeof villainSorts;

/** Notes minimales proposées (valeur d'URL → libellé). */
export const minRatingOptions = {
  "3": "3 et plus",
  "4": "4 et plus",
  "4.5": "4,5 et plus",
} as const;
type MinRating = keyof typeof minRatingOptions;

export const threatLevels: ThreatLevel[] = ["faible", "modere", "eleve", "critique"];
export const villainStatuses: VillainStatus[] = ["en-liberte", "emprisonne"];

export interface HeroFilters {
  search: string;
  sort: HeroSort;
  minRating: MinRating | undefined;
}

export interface VillainFilters {
  search: string;
  sort: VillainSort;
  threat: ThreatLevel | undefined;
  status: VillainStatus | undefined;
  /** Code du type d'incident. */
  specialty: string | undefined;
}

export interface EncyclopediaQuery {
  tab: EncyclopediaTab;
  page: number;
  heroes: HeroFilters;
  villains: VillainFilters;
}

const first = (value: string | string[] | undefined) => (Array.isArray(value) ? value[0] : value);

const pick = <T extends string>(value: string | undefined, allowed: readonly T[]) =>
  allowed.find((option) => option === value);

export function readEncyclopediaQuery(params: RawSearchParams): EncyclopediaQuery {
  const get = (name: string) => first(params[name])?.trim() || undefined;
  const p = encyclopediaParams;
  const page = Number(get(p.page));

  return {
    tab: get(p.tab) === "vilains" ? "vilains" : "heros",
    page: Number.isInteger(page) && page > 1 ? page : 1,
    heroes: {
      search: get(p.search) ?? "",
      sort: pick(get(p.sort), Object.keys(heroSorts) as HeroSort[]) ?? "alias",
      minRating: pick(get(p.minRating), Object.keys(minRatingOptions) as MinRating[]),
    },
    villains: {
      search: get(p.search) ?? "",
      sort: pick(get(p.sort), Object.keys(villainSorts) as VillainSort[]) ?? "alias",
      threat: pick(get(p.threat), threatLevels),
      status: pick(get(p.status), villainStatuses),
      specialty: get(p.specialty),
    },
  };
}

/** Sans casse ni accents, pour une recherche tolérante (« neree » trouve « Néréide »). */
const normalize = (text: string) =>
  text
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .trim();

/** Tous les mots de la recherche doivent apparaître dans l'un des champs. */
function matches(search: string, fields: string[]) {
  const words = normalize(search).split(/\s+/).filter(Boolean);
  if (words.length === 0) return true;
  const haystack = normalize(fields.join(" "));
  return words.every((word) => haystack.includes(word));
}

const byAlias = (a: { alias: string }, b: { alias: string }) =>
  a.alias.localeCompare(b.alias, "fr");

export function filterHeroes(heroes: SuperHero[], filters: HeroFilters): SuperHero[] {
  const minRating = filters.minRating ? Number(filters.minRating) : 0;
  const result = heroes.filter(
    (hero) => hero.averageRating >= minRating && matches(filters.search, [hero.alias, hero.power]),
  );
  const compare: Record<HeroSort, (a: SuperHero, b: SuperHero) => number> = {
    alias: byAlias,
    note: (a, b) => b.averageRating - a.averageRating || byAlias(a, b),
    interventions: (a, b) => b.interventionsCount - a.interventionsCount || byAlias(a, b),
  };
  return result.sort(compare[filters.sort]);
}

export function filterVillains(villains: Villain[], filters: VillainFilters): Villain[] {
  const result = villains.filter(
    (villain) =>
      (!filters.threat || villain.threatLevel === filters.threat) &&
      (!filters.status || villain.status === filters.status) &&
      (!filters.specialty || villain.specialties.some(({ code }) => code === filters.specialty)) &&
      matches(filters.search, [
        villain.alias,
        villain.firstName,
        villain.lastName,
        villain.mainSector,
        villain.description,
        ...villain.specialties.map(({ label }) => label),
      ]),
  );
  const compare: Record<VillainSort, (a: Villain, b: Villain) => number> = {
    alias: byAlias,
    menace: (a, b) =>
      threatLevels.indexOf(b.threatLevel) - threatLevels.indexOf(a.threatLevel) || byAlias(a, b),
    incidents: (a, b) => b.reportedIncidentsCount - a.reportedIncidentsCount || byAlias(a, b),
  };
  return result.sort(compare[filters.sort]);
}

/** Spécialités présentes chez au moins un vilain, par libellé (options du filtre). */
export function villainSpecialties(villains: Villain[]): VillainSpecialty[] {
  const byCode = new Map<string, VillainSpecialty>();
  for (const villain of villains) {
    for (const specialty of villain.specialties) byCode.set(specialty.code, specialty);
  }
  return [...byCode.values()].sort((a, b) => a.label.localeCompare(b.label, "fr"));
}

/** Découpe une liste filtrée en page, comme le ferait l'API. */
export function paginate<T>(items: T[], pageNumber: number, pageSize: number): Page<T> {
  const offset = (pageNumber - 1) * pageSize;
  return {
    items: items.slice(offset, offset + pageSize),
    total: items.length,
    limit: pageSize,
    offset,
  };
}

/** Paramètres d'URL des filtres actifs d'un onglet (les valeurs par défaut sont omises). */
export function filterQuery(query: EncyclopediaQuery): Record<string, string | undefined> {
  const p = encyclopediaParams;
  if (query.tab === "heros") {
    const { search, sort, minRating } = query.heroes;
    return {
      [p.search]: search || undefined,
      [p.minRating]: minRating,
      [p.sort]: sort === "alias" ? undefined : sort,
    };
  }
  const { search, sort, threat, status, specialty } = query.villains;
  return {
    [p.tab]: "vilains",
    [p.search]: search || undefined,
    [p.threat]: threat,
    [p.status]: status,
    [p.specialty]: specialty,
    [p.sort]: sort === "alias" ? undefined : sort,
  };
}

export interface ActiveFilter {
  label: string;
  /** Lien qui retire ce seul filtre (et revient à la première page). */
  href: string;
}

/** Filtres actifs de l'onglet courant, chacun avec le lien qui le retire. */
export function activeFilters(
  query: EncyclopediaQuery,
  basePath: string,
  specialties: VillainSpecialty[] = [],
): ActiveFilter[] {
  const params = filterQuery(query);
  const without = (name: string) => {
    const search = new URLSearchParams();
    for (const [key, value] of Object.entries(params)) {
      if (key !== name && value) search.set(key, value);
    }
    const string = search.toString();
    return string ? `${basePath}?${string}` : basePath;
  };
  const p = encyclopediaParams;
  const filters: ActiveFilter[] = [];

  const { search } = query.tab === "heros" ? query.heroes : query.villains;
  if (search) filters.push({ label: `« ${search} »`, href: without(p.search) });

  if (query.tab === "heros") {
    const { minRating } = query.heroes;
    if (minRating) {
      filters.push({ label: `Note : ${minRatingOptions[minRating]}`, href: without(p.minRating) });
    }
    return filters;
  }

  const { threat, status, specialty } = query.villains;
  if (threat) {
    filters.push({ label: `Menace : ${threatLevelLabels[threat]}`, href: without(p.threat) });
  }
  if (status) filters.push({ label: villainStatusLabels[status], href: without(p.status) });
  if (specialty) {
    const label = specialties.find(({ code }) => code === specialty)?.label ?? specialty;
    filters.push({ label: `Spécialité : ${label}`, href: without(p.specialty) });
  }
  return filters;
}
