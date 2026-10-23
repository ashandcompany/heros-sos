import { cache } from "react";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { accountsMock } from "@/mocks/users";
import { apiFailureMessage } from "@/lib/validation/form-state";
import type {
  AvatarUpdateResult,
  ContactUpdateResult,
  EmailChangeResult,
  HeroPosition,
  HeroPublicProfileResult,
  HeroPublicProfileUpdate,
  MutationResult,
  MyHeroProfile,
  PostalAddress,
  SuperHero,
  SuperHeroProfile,
} from "@/types";
import {
  deleteMyAvatar,
  fetchMyAccount,
  patchMyContact,
  postMyEmailChange,
  putMyAvatar,
} from "./account";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";
import { accountStatusFromApi } from "./user-moderation";

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

/**
 * Profil du super-héros connecté. API : `GET /me`, le même compte que pour un citoyen avec un bloc
 * `hero` en plus : `power` est sa description publique (`bio`). Ni la note moyenne, ni le nombre
 * d'interventions, ni la dernière intervention n'y figurent.
 */
export async function getMyHeroProfile(token: string): Promise<MyHeroProfile> {
  if (!isApiEnabled) {
    // Sans API, aucun mail n'est envoyé : une demande est confirmée d'emblée.
    const profile = await findMyProfile(token);
    return mockResponse({
      ...profile,
      pendingEmailConfirmed: profile.pendingEmail !== null,
      locationConsentAt: locationConsentsMock.get(profile.id) ?? null,
    });
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
    locationConsentAt: hero.locationConsentAt ?? null,
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
 * Change le pseudo et la description publics, aussitôt visibles sur la fiche. API : `PATCH /me` avec
 * `{ alias, bio }` ; un pseudo déjà pris (sans tenir compte de la casse) répond `409 alias_already_used`.
 */
export async function updateMyHeroPublicProfile(
  token: string,
  update: HeroPublicProfileUpdate,
): Promise<HeroPublicProfileResult> {
  if (!isApiEnabled) {
    const profile = await findMyProfile(token);
    const taken = superHeroesMock.some(
      ({ id, alias }) => id !== profile.id && alias.toLowerCase() === update.alias.toLowerCase(),
    );
    if (taken) return mockResponse({ ok: false, error: "alias-taken" });
    profile.alias = update.alias;
    profile.power = update.power;
    const hero = superHeroesMock.find(({ id }) => id === profile.id);
    if (hero) Object.assign(hero, { alias: update.alias, power: update.power });
    const account = accountsMock.find(({ user }) => user.id === profile.id);
    if (account) account.user.displayName = update.alias;
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch("/me", {
      method: "PATCH",
      sessionToken: token,
      body: { alias: update.alias, bio: update.power } satisfies ApiSchema<"ContactUpdateRequest">,
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "alias_already_used") return { ok: false, error: "alias-taken" };
      if (error.code === "validation_failed") {
        return { ok: false, error: "invalid-fields", fields: error.fields };
      }
    }
    return toApiFailure(error, "Modification du profil public");
  }
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

/** Retire la photo (profil complet et profil public). API : `DELETE /me/avatar`. */
export async function removeMyHeroAvatar(token: string): Promise<MutationResult> {
  if (isApiEnabled) return deleteMyAvatar(token);

  const profile = await findMyProfile(token);
  profile.avatarUrl = null;
  const hero = superHeroesMock.find(({ id }) => id === profile.id);
  if (hero) hero.avatarUrl = null;
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

/* ---------- Partage de position (rôle super-heros) ---------- */

/** Sans API : date du consentement au partage de position, par identifiant de héros. */
const locationConsentsMock = new Map<string, string>();

const locationErrorMessages: Record<string, string> = {
  hero_location_consent_required: "Activez d'abord le partage de position.",
  validation_failed: "Position invalide.",
};

/** Traduit l'échec d'une écriture sur la position en message affichable. */
function toLocationFailure(error: unknown, context: string): MutationResult {
  if (error instanceof ApiError && locationErrorMessages[error.code]) {
    return { ok: false, error: locationErrorMessages[error.code] };
  }
  return { ok: false, error: apiFailureMessage(toApiFailure(error, context)) };
}

async function writeLocation(
  token: string,
  path: string,
  method: "PUT" | "DELETE",
  context: string,
  body?: HeroPosition,
): Promise<MutationResult> {
  try {
    await apiFetch(path, { method, sessionToken: token, body });
    return { ok: true };
  } catch (error) {
    return toLocationFailure(error, context);
  }
}

/** Consent au partage de position (la date d'origine est conservée). API : `PUT /hero/location/consent`. */
export async function grantHeroLocationConsent(token: string): Promise<MutationResult> {
  if (!isApiEnabled) {
    const { id } = await findMyProfile(token);
    if (!locationConsentsMock.has(id)) locationConsentsMock.set(id, new Date().toISOString());
    return mockResponse({ ok: true });
  }
  return writeLocation(token, "/hero/location/consent", "PUT", "Consentement à la position");
}

/**
 * Retire le consentement ; l'API supprime aussi la position. API : `DELETE /hero/location/consent`.
 */
export async function revokeHeroLocationConsent(token: string): Promise<MutationResult> {
  if (!isApiEnabled) {
    locationConsentsMock.delete((await findMyProfile(token)).id);
    return mockResponse({ ok: true });
  }
  return writeLocation(
    token,
    "/hero/location/consent",
    "DELETE",
    "Retrait du consentement à la position",
  );
}

/**
 * Remplace la position actuelle du héros, refusée sans consentement (409). Elle n'est relue par
 * aucune route : elle sert seulement à l'alerte de proximité. API : `PUT /hero/location`.
 */
export async function saveHeroLocation(
  token: string,
  position: HeroPosition,
): Promise<MutationResult> {
  if (!isApiEnabled) {
    const { id } = await findMyProfile(token);
    if (!locationConsentsMock.has(id)) {
      return mockResponse({
        ok: false,
        error: locationErrorMessages.hero_location_consent_required,
      });
    }
    return mockResponse({ ok: true });
  }
  return writeLocation(token, "/hero/location", "PUT", "Partage de la position", position);
}

/** Supprime la position en gardant le consentement. API : `DELETE /hero/location`. */
export async function deleteHeroLocation(token: string): Promise<MutationResult> {
  if (!isApiEnabled) {
    await findMyProfile(token);
    return mockResponse({ ok: true });
  }
  return writeLocation(token, "/hero/location", "DELETE", "Suppression de la position");
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
