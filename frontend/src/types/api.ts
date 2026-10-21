/** Fenêtre demandée d'une liste paginée (`limit` : 1 à 100, 20 par défaut ; `offset` ≥ 0). */
export interface PageRequest {
  limit?: number;
  offset?: number;
}

/** Page d'une liste. Forme commune des `XxxPage` de l'OpenAPI (`items`, `total`, `limit`, `offset`). */
export interface Page<T> {
  items: T[];
  /** Nombre total d'éléments correspondant aux filtres, toutes pages confondues. */
  total: number;
  limit: number;
  offset: number;
}

/**
 * Échecs communs à tous les appels à l'API, renvoyés par `toApiFailure`. Chaque fonction de
 * `lib/api/` ajoute ses propres erreurs métier à cette union. Le 401 n'y figure pas : une session
 * expirée redirige vers la page de connexion (voir `apiFetch`).
 */
export type ApiFailure =
  /** Rôle insuffisant, origine ou jeton CSRF refusés (403). */
  | { ok: false; error: "forbidden" }
  /** Fichier ou corps trop volumineux (413). */
  | { ok: false; error: "payload-too-large" }
  /** Type de fichier ou de contenu refusé (415). */
  | { ok: false; error: "unsupported-media-type" }
  /** Trop de requêtes (429) ; délai annoncé par `Retry-After`. */
  | { ok: false; error: "too-many-attempts"; retryAfterSeconds?: number }
  /** API injoignable, délai dépassé, erreur serveur ou code inattendu. */
  | { ok: false; error: "unavailable" };
