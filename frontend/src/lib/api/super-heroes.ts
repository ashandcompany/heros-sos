import { cache } from "react";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import type {
  AvatarUpdateResult,
  ContactUpdateResult,
  EmailChangeResult,
  MyHeroProfile,
  PostalAddress,
  SuperHero,
  SuperHeroProfile,
} from "@/types";
import {
  checkEmailChangeWithMocks,
  getApiAccount,
  postEmailChangeRequest,
  putAvatar,
  toDataUrl,
  updateMyContact,
} from "./account";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Fiches publiques des super-héros (accueil, encyclopédie) : `GET /heroes` et `GET /heroes/{heroId}`,
 * publics. Seuls les héros approuvés au compte actif y figurent ; `power` est la description
 * publique du profil. Les fiches complètes du back-office restent mockées.
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

/*
 * Espace super-héros (rôle super-heros). Un héros est un compte comme un autre (`role: hero`) :
 * avec l'API, ces fonctions passent par les routes du compte (`/me`, voir `account.ts`).
 */

async function findMyProfile(token: string): Promise<SuperHeroProfile> {
  const user = await userFromToken(token, "super-heros");
  const profile = superHeroProfilesMock.find(({ id }) => id === user.id);
  if (!profile) throw new Error("404 : profil introuvable");
  return profile;
}

/**
 * Profil du héros connecté : son compte et son profil public (`hero.bio` → `power`).
 * API : `GET /me`. Un compte modérateur peut avoir une adresse vide : `address` vaut alors `null`.
 */
export async function getMyHeroProfile(token: string): Promise<MyHeroProfile> {
  if (!isApiEnabled) return mockResponse(await findMyProfile(token));

  const account = await getApiAccount(token);
  const { street, postalCode, city, country } = account.address;
  return {
    id: account.id,
    alias: account.hero?.alias ?? account.firstName,
    power: account.hero?.bio ?? "",
    avatarUrl: account.avatarUrl ?? null,
    firstName: account.firstName,
    lastName: account.lastName,
    civility: account.civility,
    birthDate: account.birthDate,
    email: account.email,
    pendingEmail: account.pendingEmail ?? null,
    mobilePhone: account.mobilePhone,
    address: street ? { street, postalCode, city, country } : null,
    registeredAt: account.createdAt,
  };
}

/**
 * Remplace le téléphone et l'adresse. Le profil d'un héros n'a pas de complément d'adresse : celui
 * du compte est conservé.
 * API : `PATCH /me` avec `{ mobilePhone, address }`.
 */
export async function updateMyHeroContact(
  token: string,
  update: { mobilePhone: string; address: PostalAddress },
): Promise<ContactUpdateResult> {
  if (isApiEnabled) {
    return updateMyContact(token, { ...update, address: { ...update.address, complement: null } });
  }

  const profile = await findMyProfile(token);
  profile.mobilePhone = update.mobilePhone;
  profile.address = update.address;
  return mockResponse({ ok: true });
}

/**
 * Remplace la photo (profil complet et profil public).
 * API : `PUT /me/avatar` (voir `putAvatar`).
 */
export async function updateMyHeroAvatar(token: string, file: File): Promise<AvatarUpdateResult> {
  if (isApiEnabled) return putAvatar(token, file);

  const profile = await findMyProfile(token);
  const avatarUrl = await toDataUrl(file);
  profile.avatarUrl = avatarUrl;
  const hero = superHeroesMock.find(({ id }) => id === profile.id);
  if (hero) hero.avatarUrl = avatarUrl;
  return mockResponse({ ok: true });
}

/**
 * Demande de changement d'email, appliquée après validation par la modération.
 * API : `POST /me/email-change-requests` avec `{ email }`.
 */
export async function requestHeroEmailChange(
  token: string,
  email: string,
): Promise<EmailChangeResult> {
  if (isApiEnabled) return postEmailChangeRequest(token, email);

  const profile = await findMyProfile(token);
  const result = checkEmailChangeWithMocks(profile.email, email);
  if (result.ok) profile.pendingEmail = email.toLowerCase();
  return mockResponse(result);
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
