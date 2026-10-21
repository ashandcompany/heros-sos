import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { accountsMock } from "@/mocks/users";
import type {
  AvatarUpdateResult,
  ContactUpdateResult,
  EmailChangeResult,
  MyHeroProfile,
  PostalAddress,
  SuperHero,
  SuperHeroProfile,
} from "@/types";
import { fetchMyAccount, patchMyContact, postMyEmailChange, putMyAvatar } from "./account";
import { isApiEnabled, mockResponse } from "./config";
import { type ApiSchema, apiFetch } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";
import { accountStatusFromApi } from "./user-moderation";

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

/** Un héros de `GET /admin/heroes` vers la fiche du back-office (rôles, statuts et dates convertis). */
function fromApiAdminHero(hero: ApiSchema<"AdminHero">): SuperHeroProfile {
  const { street, postalCode, city, country } = hero.address;
  return {
    id: hero.id,
    alias: hero.alias,
    avatarUrl: hero.avatarUrl ?? null,
    power: hero.power,
    averageRating: hero.averageRating,
    interventionsCount: hero.interventionsCount,
    firstName: hero.firstName,
    lastName: hero.lastName,
    civility: hero.civility,
    birthDate: hero.birthDate,
    email: hero.email,
    pendingEmail: hero.pendingEmail ?? null,
    mobilePhone: hero.mobilePhone,
    address: { street, postalCode, city, country },
    registeredAt: hero.createdAt,
    lastInterventionAt: hero.lastInterventionAt,
    accountStatus: accountStatusFromApi[hero.accountStatus],
  };
}

/**
 * Fiches complètes de tous les super-héros, triées par pseudo (back-office). Les données civiles
 * sont confidentielles : la route n'est ouverte qu'aux modérateurs.
 * API : `GET /admin/heroes`, page après page (100 héros au plus par page).
 */
export async function getSuperHeroProfiles(sessionToken: string): Promise<SuperHeroProfile[]> {
  if (!isApiEnabled) {
    await userFromToken(sessionToken, "admin");
    const profiles = [...superHeroProfilesMock].sort((a, b) =>
      a.alias.localeCompare(b.alias, "fr"),
    );
    return mockResponse(profiles);
  }

  const heroes: SuperHeroProfile[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(`/admin/heroes${pageQuery({ limit: MAX_PAGE_SIZE, offset })}`, {
      sessionToken,
    });
    const page = (await response.json()) as ApiSchema<"AdminHeroPage">;
    heroes.push(...page.items.map(fromApiAdminHero));
    if (page.items.length === 0 || heroes.length >= page.total) return heroes;
  }
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

/**
 * Profil du super-héros connecté. API : `GET /me`, le même compte que pour un citoyen avec un bloc
 * `hero` en plus : `power` est sa description publique (`bio`). Ni la note moyenne, ni le nombre
 * d'interventions, ni la dernière intervention n'y figurent.
 */
export async function getMyHeroProfile(token: string): Promise<MyHeroProfile> {
  if (!isApiEnabled) {
    // Sans API, aucun mail n'est envoyé : une demande est confirmée d'emblée.
    const profile = await findMyProfile(token);
    return mockResponse({ ...profile, pendingEmailConfirmed: profile.pendingEmail !== null });
  }

  const { hero, ...account } = await fetchMyAccount(token);
  if (!hero) throw new Error("403 : ce compte n'est pas un super-héros");
  const { street, postalCode, city, country } = account.address;
  return {
    id: account.id,
    alias: hero.alias,
    avatarUrl: account.avatarUrl ?? null,
    power: hero.bio,
    firstName: account.firstName,
    lastName: account.lastName,
    civility: account.civility,
    birthDate: account.birthDate,
    email: account.email,
    pendingEmail: account.pendingEmail ?? null,
    pendingEmailConfirmed: account.pendingEmailConfirmed ?? false,
    mobilePhone: account.mobilePhone,
    address: { street, postalCode, city, country },
    registeredAt: account.createdAt,
  };
}

/**
 * Modifie le téléphone et l'adresse. API : `PATCH /me`. Le profil d'un super-héros n'a pas de
 * complément d'adresse : absent de la requête, il est conservé.
 */
export async function updateMyHeroContact(
  token: string,
  update: { mobilePhone: string; address: PostalAddress },
): Promise<ContactUpdateResult> {
  if (isApiEnabled) return patchMyContact(token, update);

  const profile = await findMyProfile(token);
  profile.mobilePhone = update.mobilePhone;
  profile.address = update.address;
  return mockResponse({ ok: true });
}

/**
 * Remplace la photo (profil complet et profil public). API : `PUT /me/avatar`, le corps est l'image
 * elle-même. Elle est publique : `GET /avatars/{name}`.
 */
export async function updateMyHeroAvatar(token: string, file: File): Promise<AvatarUpdateResult> {
  if (isApiEnabled) return putMyAvatar(token, file);

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
 * API : `POST /me/email-change-requests` avec `{ email }`.
 */
export async function requestHeroEmailChange(
  token: string,
  email: string,
): Promise<EmailChangeResult> {
  if (isApiEnabled) return postMyEmailChange(token, email);

  const profile = await findMyProfile(token);
  const normalized = email.toLowerCase();

  if (normalized === profile.email) {
    return mockResponse({ ok: false, error: "email-unchanged" });
  }
  if (accountsMock.some((existing) => existing.email === normalized)) {
    return mockResponse({ ok: false, error: "email-already-used" });
  }

  profile.pendingEmail = normalized;
  return mockResponse({ ok: true });
}

/** Profil public d'un super-héros. Future implémentation : `GET ${API_URL}/heroes/${id}` (404 → `null`). */
export async function getSuperHero(id: string): Promise<SuperHero | null> {
  return mockResponse(superHeroesMock.find((hero) => hero.id === id) ?? null);
}
