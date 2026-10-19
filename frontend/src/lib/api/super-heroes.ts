import { heroApplicationsMock } from "@/mocks/hero-applications";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { accountsMock } from "@/mocks/users";
import type { HeroApplication, MutationResult, SuperHero, SuperHeroProfile } from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

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
 * Future implémentation : `GET ${API_URL}/admin/heroes` avec `Authorization: Bearer ${sessionToken}`.
 */
export async function getSuperHeroProfiles(sessionToken: string): Promise<SuperHeroProfile[]> {
  await userFromToken(sessionToken, "admin");
  const profiles = [...superHeroProfilesMock].sort((a, b) => a.alias.localeCompare(b.alias, "fr"));
  return mockResponse(profiles);
}

/**
 * Tous les super-héros (profil public), triés par pseudo, pour l'encyclopédie.
 * Future implémentation : `GET ${API_URL}/heroes?sort=alias`
 */
export async function getSuperHeroes(): Promise<SuperHero[]> {
  return mockResponse([...superHeroesMock].sort((a, b) => a.alias.localeCompare(b.alias, "fr")));
}

/**
 * Demandes d'inscription de super-héros, les plus anciennes d'abord (ordre de traitement).
 * Future implémentation : `GET ${API_URL}/admin/hero-applications?status=${status}`
 */
export async function getHeroApplications(
  token: string,
  status?: HeroApplication["status"],
): Promise<HeroApplication[]> {
  await userFromToken(token, "admin");
  return mockResponse(
    heroApplicationsMock
      .filter((application) => !status || application.status === status)
      .sort((a, b) => Date.parse(a.submittedAt) - Date.parse(b.submittedAt)),
  );
}

/**
 * Valide ou refuse une demande. Une demande validée crée le profil du super-héros ; l'API enverra
 * au candidat un email pour choisir son mot de passe.
 * Future implémentation : `POST ${API_URL}/admin/hero-applications/${id}/${approve ? "approve" : "reject"}`
 */
export async function reviewHeroApplication(
  token: string,
  id: string,
  approve: boolean,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const application = heroApplicationsMock.find((existing) => existing.id === id);
  if (!application || application.status !== "en-attente") {
    return mockResponse({ ok: false, error: "Cette demande a déjà été traitée." });
  }
  if (approve && superHeroesMock.some(({ alias }) => alias === application.alias)) {
    return mockResponse({ ok: false, error: "Ce pseudo est déjà utilisé par un super-héros." });
  }

  const now = new Date().toISOString();
  application.status = approve ? "valide" : "rejete";
  application.reviewedAt = now;

  if (approve) {
    const heroId = `hero-${crypto.randomUUID()}`;
    const hero: SuperHero = {
      id: heroId,
      alias: application.alias,
      avatarUrl: null,
      power: application.power,
      averageRating: 0,
      interventionsCount: 0,
    };
    superHeroesMock.push(hero);
    superHeroProfilesMock.push({
      ...hero,
      firstName: application.firstName,
      lastName: application.lastName,
      civility: application.civility,
      birthDate: application.birthDate,
      email: application.email,
      mobilePhone: application.mobilePhone,
      address: application.address,
      registeredAt: now,
      lastInterventionAt: null,
    });
    accountsMock.push({
      user: { id: heroId, displayName: application.alias, role: "super-heros" },
      email: application.email,
      // Mock : mot de passe aléatoire, inutilisable tant que le héros ne l'a pas redéfini.
      password: crypto.randomUUID(),
    });
  }

  return mockResponse({ ok: true });
}
