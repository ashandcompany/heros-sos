import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import type { SuperHero, SuperHeroProfile } from "@/types";
import { mockResponse } from "./config";

/**
 * Super-héros mis en avant sur l'accueil (les mieux notés).
 * Future implémentation : `GET ${API_URL}/heroes?sort=-averageRating&limit=${limit}`
 */
export async function getFeaturedSuperHeroes(limit = 8): Promise<SuperHero[]> {
  const featured = [...superHeroesMock]
    .sort((a, b) => b.averageRating - a.averageRating)
    .slice(0, limit);

  return mockResponse(featured);
}

/**
 * Fiches complètes de tous les super-héros, triées par pseudo (back-office).
 * Le jeton de session sera transmis à l'API, qui doit vérifier le rôle admin.
 * Future implémentation : `GET ${API_URL}/admin/heroes` avec `Authorization: Bearer ${sessionToken}`.
 */
export async function getSuperHeroProfiles(sessionToken: string): Promise<SuperHeroProfile[]> {
  void sessionToken;
  const profiles = [...superHeroProfilesMock].sort((a, b) => a.alias.localeCompare(b.alias, "fr"));
  return mockResponse(profiles);
}
