import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { accountsMock } from "@/mocks/users";
import type { MutationResult, PostalAddress, SuperHero, SuperHeroProfile } from "@/types";
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

/** Profil public d'un super-héros. Future implémentation : `GET ${API_URL}/heroes/${id}` (404 → `null`). */
export async function getSuperHero(id: string): Promise<SuperHero | null> {
  return mockResponse(superHeroesMock.find((hero) => hero.id === id) ?? null);
}
