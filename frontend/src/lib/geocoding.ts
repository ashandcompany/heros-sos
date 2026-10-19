/**
 * Autocomplétion d'adresses avec les données OpenStreetMap, via le géocodeur Photon.
 * Nominatim (le géocodeur officiel d'OSM) interdit l'autocomplétion dans sa politique
 * d'utilisation ; Photon est conçu pour la recherche au fil de la frappe.
 *
 * À n'importer que côté serveur (route /api/adresses) : l'adresse IP des visiteurs n'est pas
 * transmise au service. L'instance publique de Komoot est soumise à un usage raisonnable :
 * en production, prévoir une instance Photon dédiée (PHOTON_URL) ou un relais par l'API Go.
 */

const PHOTON_URL = process.env.PHOTON_URL || "https://photon.komoot.io/api/";
const USER_AGENT = "Heros-SOS (https://github.com/ashandcompany/heros-sos)";
const RESULT_LIMIT = 6;
const TIMEOUT_MS = 4000;

/** `postal` : adresses postales (numéro et rue). `place` : tout lieu (commerce, gare, rue…). */
export type AddressSearchKind = "postal" | "place";

export interface AddressSuggestion {
  id: string;
  /** Texte principal, recopié dans le champ à la sélection. */
  label: string;
  /** Précision affichée sous le libellé (code postal, ville, pays). */
  context: string;
  street: string;
  postalCode: string;
  city: string;
  /** Code pays ISO 3166-1 alpha-2, ou chaîne vide si inconnu. */
  country: string;
}

interface PhotonProperties {
  osm_type?: string;
  osm_id?: number;
  type?: string;
  name?: string;
  housenumber?: string;
  street?: string;
  postcode?: string;
  city?: string;
  locality?: string;
  district?: string;
  country?: string;
  countrycode?: string;
}

function toSuggestion(properties: PhotonProperties): AddressSuggestion | null {
  const streetLine = properties.street
    ? [properties.housenumber, properties.street].filter(Boolean).join(" ")
    : properties.type === "street"
      ? (properties.name ?? "")
      : "";
  // Un lieu nommé (gare, commerce…) garde son nom : c'est ce que l'utilisateur a cherché.
  const placeName = properties.type !== "street" && !properties.housenumber ? properties.name : "";
  const label = [placeName, streetLine].filter(Boolean).join(", ");
  const city = properties.city ?? properties.locality ?? properties.district ?? "";

  if (!label) return null;

  return {
    id: `${properties.osm_type ?? ""}${properties.osm_id ?? label}`,
    label,
    context: [properties.postcode, city, properties.country].filter(Boolean).join(" · "),
    street: streetLine || label,
    postalCode: properties.postcode ?? "",
    city,
    country: properties.countrycode?.toUpperCase() ?? "",
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

  const url = new URL(PHOTON_URL);
  url.searchParams.set("q", query);
  url.searchParams.set("lang", "fr");
  url.searchParams.set("limit", String(RESULT_LIMIT * 2));
  if (kind === "postal") {
    url.searchParams.append("layer", "house");
    url.searchParams.append("layer", "street");
  }

  const response = await fetch(url, {
    headers: { "User-Agent": USER_AGENT, Accept: "application/json" },
    signal: AbortSignal.timeout(TIMEOUT_MS),
  });
  if (!response.ok) throw new Error(`Photon : HTTP ${response.status}`);

  const { features = [] } = (await response.json()) as {
    features?: Array<{ properties?: PhotonProperties }>;
  };

  // OSM contient des doublons (même adresse sur plusieurs objets) : on garde le premier.
  const seen = new Set<string>();
  const suggestions = features
    .map(({ properties }) => (properties ? toSuggestion(properties) : null))
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
