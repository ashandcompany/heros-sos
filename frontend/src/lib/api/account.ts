import { accountsMock } from "@/mocks/users";
import { citizensMock } from "@/mocks/citizens";
import type {
  CitizenAccount,
  ContactUpdate,
  ModerationStatus,
  MutationResult,
  MyAccount,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { getApiSession } from "./http";
import { userFromToken } from "./session-user";

async function findMyAccount(token: string): Promise<CitizenAccount> {
  const user = await userFromToken(token, "citoyen");
  const account = citizensMock.find(({ id }) => id === user.id);
  if (!account) throw new Error("404 : compte introuvable");
  return account;
}

/**
 * Compte du citoyen connecté. API : `GET /auth/me` (prénom et email, sans nouvel appel : la
 * session est déjà lue pour ce rendu) ; le reste du profil attend `GET ${API_URL}/me`, absent du
 * contrat (`profile: null`). Mock : le profil complet.
 */
export async function getMyAccount(token: string): Promise<MyAccount> {
  if (isApiEnabled) {
    const session = await getApiSession(token);
    if (!session) throw new Error("401 : session invalide");
    return { firstName: session.user.displayName, email: session.user.email, profile: null };
  }

  const profile = await findMyAccount(token);
  return mockResponse({ firstName: profile.firstName, email: profile.email, profile });
}

/** Future implémentation : `PATCH ${API_URL}/me` avec `{ mobilePhone, address }`. */
export async function updateMyContact(
  token: string,
  update: ContactUpdate,
): Promise<MutationResult> {
  const account = await findMyAccount(token);
  account.mobilePhone = update.mobilePhone;
  account.address = update.address;
  return mockResponse({ ok: true });
}

/**
 * Remplace la photo de profil.
 * Future implémentation : `PUT ${API_URL}/me/avatar` en multipart/form-data. Le mock garde l'image
 * en mémoire sous forme de data URL.
 */
export async function updateMyAvatar(token: string, file: File): Promise<MutationResult> {
  const account = await findMyAccount(token);
  const base64 = Buffer.from(await file.arrayBuffer()).toString("base64");
  account.avatarUrl = `data:${file.type};base64,${base64}`;
  return mockResponse({ ok: true });
}

/**
 * Demande de changement d'email : l'adresse n'est remplacée qu'après validation par la modération.
 * Future implémentation : `POST ${API_URL}/me/email-change-requests` avec `{ email }`.
 */
export async function requestEmailChange(token: string, email: string): Promise<MutationResult> {
  const account = await findMyAccount(token);
  const normalized = email.toLowerCase();

  if (normalized === account.email) {
    return mockResponse({ ok: false, error: "C'est déjà votre adresse email actuelle." });
  }
  if (accountsMock.some((existing) => existing.email === normalized)) {
    return mockResponse({
      ok: false,
      error: "Cette adresse est déjà utilisée par un autre compte.",
    });
  }

  account.pendingEmail = normalized;
  return mockResponse({ ok: true });
}

/** Future implémentation : `PUT ${API_URL}/me/newsletter` avec `{ subscribed }`. */
export async function updateNewsletterPreference(
  token: string,
  subscribed: boolean,
): Promise<MutationResult> {
  const account = await findMyAccount(token);
  account.newsletterSubscribed = subscribed;
  return mockResponse({ ok: true });
}

/* ---------- Back-office (rôle admin) ---------- */

/**
 * Comptes citoyens, des plus récents aux plus anciens.
 * Future implémentation : `GET ${API_URL}/admin/citizens?sort=-registeredAt`
 */
export async function getCitizenAccounts(token: string): Promise<CitizenAccount[]> {
  await userFromToken(token, "admin");
  return mockResponse(
    [...citizensMock].sort((a, b) => Date.parse(b.registeredAt) - Date.parse(a.registeredAt)),
  );
}

/**
 * Valide un compte nouvellement créé, ou le suspend (`rejete` : connexion impossible).
 * Future implémentation : `PUT ${API_URL}/admin/citizens/${id}/moderation` avec `{ decision }`.
 */
export async function moderateAccount(
  token: string,
  id: string,
  decision: ModerationStatus,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const account = citizensMock.find((existing) => existing.id === id);
  if (!account) return mockResponse({ ok: false, error: "Compte introuvable." });

  account.moderation = decision;
  return mockResponse({ ok: true });
}

/**
 * Accepte ou refuse le changement d'adresse email demandé par un citoyen.
 * Future implémentation : `POST ${API_URL}/admin/citizens/${id}/email-change/${approve ? "approve" : "reject"}`
 */
export async function reviewEmailChange(
  token: string,
  id: string,
  approve: boolean,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const account = citizensMock.find((existing) => existing.id === id);
  if (!account?.pendingEmail) {
    return mockResponse({ ok: false, error: "Aucune demande de changement en attente." });
  }

  if (approve) {
    const login = accountsMock.find(({ user }) => user.id === id);
    if (login) login.email = account.pendingEmail;
    account.email = account.pendingEmail;
  }
  account.pendingEmail = null;
  return mockResponse({ ok: true });
}
