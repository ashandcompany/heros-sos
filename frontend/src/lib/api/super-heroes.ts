import { superHeroesMock } from "@/mocks/super-heroes";
import type { SuperHero } from "@/types";
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
