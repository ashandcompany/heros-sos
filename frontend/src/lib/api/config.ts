/**
 * URL de base de l'API REST Go, préfixe compris (ex : `http://localhost:8080/api/v1`).
 * Lue côté serveur uniquement, au démarrage : tous les appels partent des Server Components,
 * des Server Actions et de `proxy.ts`, jamais du navigateur.
 *
 * Vide, l'application tourne entièrement sur les mocks. Renseignée, les fonctions déjà branchées
 * (connexion, session, déconnexion : voir docs/backend-contract.md) appellent l'API ; les autres
 * restent sur leurs mocks.
 */
export const API_URL = (process.env.API_URL ?? "").replace(/\/+$/, "");

/** Vrai quand l'API Go est configurée : les fonctions branchées l'appellent au lieu des mocks. */
export const isApiEnabled = API_URL !== "";

/** Simule un appel réseau asynchrone en renvoyant une copie des données mockées. */
export async function mockResponse<T>(data: T): Promise<T> {
  return structuredClone(data);
}
