import { accountsMock } from "@/mocks/users";
import { citizensMock } from "@/mocks/citizens";
import type { CitizenAccount, ContactUpdate, MutationResult } from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

async function findMyAccount(token: string): Promise<CitizenAccount> {
  const user = await userFromToken(token, "citoyen");
  const account = citizensMock.find(({ id }) => id === user.id);
  if (!account) throw new Error("404 : compte introuvable");
  return account;
}

/** Future implémentation : `GET ${API_URL}/me` */
export async function getMyAccount(token: string): Promise<CitizenAccount> {
  return mockResponse(await findMyAccount(token));
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
