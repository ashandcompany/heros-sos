import { cache } from "react";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { accountsMock } from "@/mocks/users";
import type { MutationResult, PostalAddress, SuperHero, SuperHeroProfile } from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Fiches publiques des super-héros (accueil, encyclopédie) : `GET /heroes` et `GET /heroes/{heroId}`,
 * publics. Seuls les héros approuvés au compte actif y figurent ; `power` est la description
 * publique du profil. Les fiches complètes (back-office, espace héros) restent mockées.
 */

function fromApiHero(hero: ApiSchema<"PublicHero">): SuperHero {
  return {
    id: hero.id,
    alias: hero.alias,
    avatarUrl: hero.avatarUrl,
    power: hero.power,
    averageRating: hero.averageRating,
    interventionsCount: hero.interventionsCount,
  };
}

const byAlias = (a: SuperHero, b: SuperHero) => a.alias.localeCompare(b.alias, "fr");

/**
 * Super-héros mis en avant sur l'accueil (les mieux notés).
 * API : `GET /heroes?sort=-averageRating&limit=…`.
 */
export async function getFeaturedSuperHeroes(limit = 8): Promise<SuperHero[]> {
  if (!isApiEnabled) {
    const featured = [...superHeroesMock]
      .sort((a, b) => b.averageRating - a.averageRating)
      .slice(0, limit);
    return mockResponse(featured);
  }

  const response = await apiFetch(`/heroes${pageQuery({ limit }, { sort: "-averageRating" })}`);
  return ((await response.json()) as ApiSchema<"PublicHeroPage">).items.map(fromApiHero);
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
 * API : `GET /heroes` (tri par pseudo par défaut), page après page (100 héros au plus par page).
 */
export async function getSuperHeroes(): Promise<SuperHero[]> {
  if (!isApiEnabled) return mockResponse([...superHeroesMock].sort(byAlias));

  const heroes: SuperHero[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(`/heroes${pageQuery({ limit: MAX_PAGE_SIZE, offset })}`);
    const page = (await response.json()) as ApiSchema<"PublicHeroPage">;
    heroes.push(...page.items.map(fromApiHero));
    if (page.items.length === 0 || heroes.length >= page.total) return heroes.sort(byAlias);
  }
}

/* ---------- Espace super-héros (rôle super-heros) ---------- */

async function findMyProfile(token: string): Promise<SuperHeroProfile> {
  const user = await userFromToken(token, "super-heros");
  const profile = superHeroProfilesMock.find(({ id }) => id === user.id);
  if (!profile) throw new Error("404 : profil introuvable");
  return profile;
}

/** Future implémentation : `GET ${API_URL}/hero/me` */
export async function getMyHeroProfile(token: string): Promise<SuperHeroProfile> {
  return mockResponse(await findMyProfile(token));
}

/** Future implémentation : `PATCH ${API_URL}/hero/me` avec `{ mobilePhone, address }`. */
export async function updateMyHeroContact(
  token: string,
  update: { mobilePhone: string; address: PostalAddress },
): Promise<MutationResult> {
  const profile = await findMyProfile(token);
  profile.mobilePhone = update.mobilePhone;
  profile.address = update.address;
  return mockResponse({ ok: true });
}

/**
 * Remplace la photo (profil complet et profil public).
 * Future implémentation : `PUT ${API_URL}/hero/me/avatar` en multipart/form-data.
 */
export async function updateMyHeroAvatar(token: string, file: File): Promise<MutationResult> {
  const profile = await findMyProfile(token);
  const base64 = Buffer.from(await file.arrayBuffer()).toString("base64");
  const avatarUrl = `data:${file.type};base64,${base64}`;
  profile.avatarUrl = avatarUrl;
  const hero = superHeroesMock.find(({ id }) => id === profile.id);
  if (hero) hero.avatarUrl = avatarUrl;
  return mockResponse({ ok: true });
}

/**
 * Demande de changement d'email, appliquée après validation par la modération.
 * Future implémentation : `POST ${API_URL}/hero/me/email-change-requests` avec `{ email }`.
 */
export async function requestHeroEmailChange(
  token: string,
  email: string,
): Promise<MutationResult> {
  const profile = await findMyProfile(token);
  const normalized = email.toLowerCase();

  if (normalized === profile.email) {
    return mockResponse({ ok: false, error: "C'est déjà votre adresse email actuelle." });
  }
  if (accountsMock.some((existing) => existing.email === normalized)) {
    return mockResponse({
      ok: false,
      error: "Cette adresse est déjà utilisée par un autre compte.",
    });
  }

  profile.pendingEmail = normalized;
  return mockResponse({ ok: true });
}

/**
 * Profil public d'un super-héros, `null` s'il n'existe pas ou n'est pas public. Mémorisé le temps
 * d'un rendu (métadonnées et page le relisent).
 * API : `GET /heroes/{heroId}` (404, y compris pour un identifiant mal formé → `null`).
 */
export const getSuperHero = cache(async (id: string): Promise<SuperHero | null> => {
  if (!isApiEnabled) return mockResponse(superHeroesMock.find((hero) => hero.id === id) ?? null);

  try {
    const response = await apiFetch(`/heroes/${encodeURIComponent(id)}`);
    return fromApiHero((await response.json()) as ApiSchema<"PublicHero">);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
});
