/**
 * Fond de carte des cartes Leaflet. Lu côté serveur puis transmis au composant client : on peut
 * changer de fournisseur au démarrage du conteneur, sans reconstruire l'image.
 *
 * Par défaut, les tuiles publiques d'OpenStreetMap. Leur politique d'utilisation
 * (https://operations.osmfoundation.org/policies/tiles/) impose de garder l'attribution visible,
 * d'envoyer un en-tête Referer (« Referrer-Policy: strict-origin-when-cross-origin » dans Caddy
 * convient) et interdit l'usage intensif et le téléchargement en masse : elles conviennent au
 * développement et à un faible trafic. Au-delà, renseigner MAP_TILE_URL avec un autre fournisseur,
 * par exemple le Plan IGN de la Géoplateforme (gratuit, sans clé) :
 *   https://data.geopf.fr/wmts?SERVICE=WMTS&REQUEST=GetTile&VERSION=1.0.0&LAYER=GEOGRAPHICALGRIDSYSTEMS.PLANIGNV2&STYLE=normal&TILEMATRIXSET=PM&FORMAT=image/png&TILEMATRIX={z}&TILEROW={y}&TILECOL={x}
 *   avec MAP_TILE_ATTRIBUTION="© IGN – Plan IGN" et MAP_TILE_MAX_ZOOM=19.
 */

export interface MapTileConfig {
  /** Modèle d'URL Leaflet (`{z}`, `{x}`, `{y}`, `{s}`). */
  url: string;
  /** Attribution affichée en bas de la carte (HTML autorisé par Leaflet : texte de confiance). */
  attribution: string;
  maxZoom: number;
}

const DEFAULT_TILES: MapTileConfig = {
  url: "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
  attribution:
    '&copy; <a href="https://www.openstreetmap.org/copyright">Contributeurs OpenStreetMap</a>',
  maxZoom: 19,
};

export function getMapTileConfig(): MapTileConfig {
  const url = process.env.MAP_TILE_URL?.trim();
  if (!url) return DEFAULT_TILES;

  const maxZoom = Number(process.env.MAP_TILE_MAX_ZOOM);
  return {
    url,
    attribution: process.env.MAP_TILE_ATTRIBUTION?.trim() ?? "",
    maxZoom: Number.isInteger(maxZoom) && maxZoom > 0 ? maxZoom : DEFAULT_TILES.maxZoom,
  };
}
