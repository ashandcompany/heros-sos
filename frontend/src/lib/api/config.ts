/**
 * URL de base de l'API REST Go. Vide tant que l'API n'existe pas :
 * les fonctions de `src/lib/api/` renvoient alors les données mockées.
 */
export const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "";

/** Simule un appel réseau asynchrone en renvoyant une copie des données mockées. */
export async function mockResponse<T>(data: T): Promise<T> {
  return structuredClone(data);
}
