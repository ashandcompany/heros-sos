/**
 * Autocomplétion d'adresses avec la Base Adresse Nationale, via le service d'autocomplétion de la
 * Géoplateforme de l'IGN : gratuit, sans clé, France métropolitaine et outre-mer.
 * Documentation : https://cartes.gouv.fr/aide/fr/guides-utilisateur/utiliser-les-services-de-la-geoplateforme/autocompletion/
 *
 * À n'importer que côté serveur (route /api/adresses) : l'adresse IP des visiteurs n'est pas
 * transmise au service, et le cache ci-dessous, commun à tous les visiteurs, aide à rester sous
 * la limite annoncée (10 requêtes par seconde et par adresse IP).
 */

const COMPLETION_URL = "https://data.geopf.fr/geocodage/completion/";
const RESULT_LIMIT = 6;
const TIMEOUT_MS = 4000;

/** `postal` : adresses postales (numéro et rue). `place` : tout lieu (gare, monument, rue…). */
export type AddressSearchKind = "postal" | "place";

export interface AddressSuggestion {
  id: string;
  /** Texte principal, recopié dans le champ à la sélection. */
  label: string;
  /** Précision affichée sous le libellé (code postal, ville). */
  context: string;
  street: string;
  postalCode: string;
  city: string;
  /** Code pays ISO 3166-1 alpha-2 : toujours `FR`, outre-mer compris. */
  country: string;
}

interface CompletionResult {
  /** Malgré son nom, le type de résultat : `StreetAddress` ou `PositionOfInterest`. */
  country?: string;
  /** `housenumber`, `street`, `locality`… pour une adresse ; nature du lieu sinon. */
  kind?: string;
  /** Nom d'un lieu (`PositionOfInterest`). */
  names?: string[];
  street?: string;
  zipcode?: string;
  city?: string;
  /** Ex : `12 Rue Victor Hugo, 76000 Rouen`. */
  fulltext?: string;
  x?: number;
  y?: number;
}

function toSuggestion(result: CompletionResult): AddressSuggestion | null {
  const { fulltext = "", zipcode = "", city = "" } = result;
  // Le numéro n'a pas de champ à lui : la première partie du texte complet donne « 12 Rue … ».
  const separator = fulltext.lastIndexOf(", ");
  const streetLine = separator > 0 ? fulltext.slice(0, separator) : (result.street ?? "");
  const placeName = result.country === "PositionOfInterest" ? result.names?.[0] : undefined;
  const label = placeName ?? streetLine;

  if (!label) return null;

  return {
    id: `${result.kind ?? ""}:${fulltext}:${result.x ?? ""},${result.y ?? ""}`,
    label,
    context: [zipcode, city].filter(Boolean).join(" · "),
    street: streetLine || label,
    postalCode: zipcode,
    city,
    country: "FR",
  };
}

/* Cache mémoire : la même saisie revient souvent (retour arrière, plusieurs utilisateurs). */
const CACHE_TTL_MS = 10 * 60 * 1000;
const CACHE_MAX_ENTRIES = 500;
const cache = new Map<string, { expiresAt: number; suggestions: AddressSuggestion[] }>();

export async function searchAddresses(
  query: string,
  kind: AddressSearchKind,
): Promise<AddressSuggestion[]> {
  const key = `${kind}:${query.toLowerCase()}`;
  const cached = cache.get(key);
  if (cached && cached.expiresAt > Date.now()) return cached.suggestions;

  const url = new URL(COMPLETION_URL);
  url.searchParams.set("text", query);
  url.searchParams.set(
    "type",
    kind === "postal" ? "StreetAddress" : "StreetAddress,PositionOfInterest",
  );
  url.searchParams.set("maximumResponses", String(RESULT_LIMIT * 2));

  const response = await fetch(url, {
    headers: { Accept: "application/json" },
    signal: AbortSignal.timeout(TIMEOUT_MS),
  });
  if (!response.ok) throw new Error(`Géoplateforme : HTTP ${response.status}`);

  const { results = [] } = (await response.json()) as { results?: CompletionResult[] };

  // Un même lieu peut revenir plusieurs fois (codes postaux, anciennes communes) : on garde le premier.
  const seen = new Set<string>();
  const suggestions = results
    .map(toSuggestion)
    .filter((suggestion): suggestion is AddressSuggestion => {
      if (!suggestion) return false;
      const signature = `${suggestion.label}|${suggestion.context}`;
      if (seen.has(signature)) return false;
      seen.add(signature);
      return true;
    })
    .slice(0, RESULT_LIMIT);

  if (cache.size >= CACHE_MAX_ENTRIES) cache.delete(cache.keys().next().value!);
  cache.set(key, { expiresAt: Date.now() + CACHE_TTL_MS, suggestions });
  return suggestions;
}
