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
