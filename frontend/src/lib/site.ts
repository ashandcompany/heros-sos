/**
 * URL publique du site, sans slash final (ex : `https://heros-sos.fr`) : liens absolus du sitemap,
 * de robots.txt et des métadonnées Open Graph. Lue côté serveur, au démarrage, comme `API_URL` :
 * la même image Docker sert dans tous les environnements.
 *
 * Toute route qui l'utilise doit être rendue à la requête (`await connection()`) : lue pendant le
 * build, l'URL serait figée dans la page prérendue.
 */
export function getSiteUrl(): string {
  return (process.env.SITE_URL?.trim() || "http://localhost:3000").replace(/\/+$/, "");
}

/**
 * Nom de domaine du site, sans port (ex : `heros-sos.fr`) : celui sous lequel un gestionnaire de mots de passe
 * enregistre la connexion. `undefined` si `SITE_URL` n'est pas une URL : un réglage faux ne doit pas casser l'appelant.
 */
export function getSiteDomain(): string | undefined {
  try {
    return new URL(getSiteUrl()).hostname;
  } catch {
    return undefined;
  }
}
