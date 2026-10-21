import { cache } from "react";
import { accountsMock } from "@/mocks/users";
import { citizensMock } from "@/mocks/citizens";
import type {
  AvatarUpdateResult,
  CitizenAccount,
  ContactUpdate,
  ContactUpdateResult,
  EmailChangeRequest,
  EmailChangeResult,
  EmailChangeReviewResult,
  MyAccount,
  NewsletterPreferenceResult,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Compte de l'utilisateur connecté : `GET /me`, `PATCH /me`, `PUT /me/avatar`,
 * `POST /me/email-change-requests`, `PUT /me/newsletter`. Un super-héros utilise les mêmes routes
 * que le citoyen (même compte, `role: hero`) : les fonctions de son espace (`super-heroes.ts`)
 * délèguent à celles-ci avec l'API.
 */

/**
 * Profil brut du compte connecté. Mémorisé le temps d'un rendu : la page de profil et la photo de
 * la barre de navigation ne déclenchent qu'un seul `GET /me`.
 */
export const getApiAccount = cache(async (token: string): Promise<ApiSchema<"Account">> => {
  const response = await apiFetch("/me", { sessionToken: token });
  return (await response.json()) as ApiSchema<"Account">;
});

function fromApiAccount(account: ApiSchema<"Account">): MyAccount {
  return {
    id: account.id,
    civility: account.civility,
    firstName: account.firstName,
    lastName: account.lastName,
    birthDate: account.birthDate,
    email: account.email,
    pendingEmail: account.pendingEmail ?? null,
    mobilePhone: account.mobilePhone,
    address: { ...account.address, complement: account.address.complement ?? null },
    avatarUrl: account.avatarUrl ?? null,
    newsletterSubscribed: account.newsletterSubscribed,
    registeredAt: account.createdAt,
  };
}

async function findMyAccount(token: string): Promise<CitizenAccount> {
  const user = await userFromToken(token, "citoyen");
  const account = citizensMock.find(({ id }) => id === user.id);
  if (!account) throw new Error("404 : compte introuvable");
  return account;
}

/** Compte du citoyen connecté. API : `GET /me`. */
export async function getMyAccount(token: string): Promise<MyAccount> {
  if (isApiEnabled) return fromApiAccount(await getApiAccount(token));
  return mockResponse(await findMyAccount(token));
}

/**
 * Remplace le téléphone et l'adresse. `address.complement` à `null` conserve le complément actuel,
 * une chaîne vide l'efface.
 * API : `PATCH /me` avec `{ mobilePhone, address }`.
 */
export async function updateMyContact(
  token: string,
  update: ContactUpdate,
): Promise<ContactUpdateResult> {
  if (!isApiEnabled) {
    const account = await findMyAccount(token);
    const complement = update.address.complement ?? account.address.complement;
    account.mobilePhone = update.mobilePhone;
    account.address = { ...update.address, complement: complement || null };
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch("/me", {
      method: "PATCH",
      sessionToken: token,
      body: update satisfies ApiSchema<"ContactUpdateRequest">,
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError && error.code === "validation_failed") {
      return { ok: false, error: "invalid-fields", fields: error.fields };
    }
    return toApiFailure(error, "Modification des coordonnées");
  }
}

/**
 * Remplace la photo de profil (publique pour un super-héros). L'API ré-encode l'image en JPEG et
 * lui donne une nouvelle adresse à chaque envoi.
 * API : `PUT /me/avatar`, l'image brute en corps (ni JSON, ni multipart) avec son `Content-Type`.
 * Mock : l'image est gardée en mémoire sous forme de data URL.
 */
export async function updateMyAvatar(token: string, file: File): Promise<AvatarUpdateResult> {
  if (!isApiEnabled) {
    const account = await findMyAccount(token);
    account.avatarUrl = await toDataUrl(file);
    return mockResponse({ ok: true });
  }

  return putAvatar(token, file);
}

/** `PUT /me/avatar`, commun au citoyen et au super-héros. */
export async function putAvatar(token: string, file: File): Promise<AvatarUpdateResult> {
  try {
    await apiFetch("/me/avatar", {
      method: "PUT",
      sessionToken: token,
      rawBody: file,
      headers: { "Content-Type": file.type },
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError && error.code === "validation_failed") {
      return {
        ok: false,
        error: error.fields.avatar === "too_large" ? "image-too-large" : "invalid-image",
      };
    }
    return toApiFailure(error, "Photo de profil");
  }
}

/** Mock : l'image sous forme de data URL. */
export async function toDataUrl(file: File): Promise<string> {
  const base64 = Buffer.from(await file.arrayBuffer()).toString("base64");
  return `data:${file.type};base64,${base64}`;
}

/**
 * Demande de changement d'email : l'adresse n'est remplacée qu'après validation par la modération.
 * Une nouvelle demande remplace la précédente.
 * API : `POST /me/email-change-requests` avec `{ email }`.
 */
export async function requestEmailChange(token: string, email: string): Promise<EmailChangeResult> {
  if (!isApiEnabled) {
    const account = await findMyAccount(token);
    const result = checkEmailChangeWithMocks(account.email, email);
    if (result.ok) account.pendingEmail = email.toLowerCase();
    return mockResponse(result);
  }

  return postEmailChangeRequest(token, email);
}

/** `POST /me/email-change-requests`, commun au citoyen et au super-héros. */
export async function postEmailChangeRequest(
  token: string,
  email: string,
): Promise<EmailChangeResult> {
  try {
    await apiFetch("/me/email-change-requests", {
      method: "POST",
      sessionToken: token,
      body: { email } satisfies ApiSchema<"EmailChangeRequestPayload">,
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "email_unchanged") return { ok: false, error: "email-unchanged" };
      if (error.code === "email_already_used") return { ok: false, error: "email-already-used" };
      if (error.code === "validation_failed") return { ok: false, error: "invalid-email" };
    }
    return toApiFailure(error, "Demande de changement d'email");
  }
}

/** Mock : mêmes refus que l'API (adresse inchangée, ou déjà prise par un autre compte). */
export function checkEmailChangeWithMocks(currentEmail: string, email: string): EmailChangeResult {
  const normalized = email.toLowerCase();
  if (normalized === currentEmail) return { ok: false, error: "email-unchanged" };
  if (accountsMock.some((existing) => existing.email === normalized)) {
    return { ok: false, error: "email-already-used" };
  }
  return { ok: true };
}

/** API : `PUT /me/newsletter` avec `{ subscribed }` (idempotent, sans mail de confirmation). */
export async function updateNewsletterPreference(
  token: string,
  subscribed: boolean,
): Promise<NewsletterPreferenceResult> {
  if (!isApiEnabled) {
    const account = await findMyAccount(token);
    account.newsletterSubscribed = subscribed;
    return mockResponse({ ok: true });
  }

  try {
    await apiFetch("/me/newsletter", {
      method: "PUT",
      sessionToken: token,
      body: { subscribed } satisfies ApiSchema<"NewsletterPreferencePayload">,
    });
    return { ok: true };
  } catch (error) {
    return toApiFailure(error, "Préférence de newsletter");
  }
}

/* ---------- Back-office (rôle admin) ---------- */

function fromApiEmailChange(item: ApiSchema<"EmailChangeRequestItem">): EmailChangeRequest {
  return {
    id: item.id,
    userId: item.userId,
    firstName: item.firstName,
    lastName: item.lastName,
    currentEmail: item.currentEmail,
    newEmail: item.newEmail,
    submittedAt: item.createdAt,
  };
}

/**
 * Changements d'adresse email en attente de validation, les plus anciens d'abord.
 * API : `GET /admin/email-change-requests?status=pending`, page après page.
 */
export async function getEmailChangeRequests(token: string): Promise<EmailChangeRequest[]> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    return mockResponse(
      citizensMock.flatMap((account) =>
        account.pendingEmail
          ? [
              {
                id: account.id,
                userId: account.id,
                firstName: account.firstName,
                lastName: account.lastName,
                currentEmail: account.email,
                newEmail: account.pendingEmail,
                submittedAt: account.registeredAt,
              },
            ]
          : [],
      ),
    );
  }

  const requests: EmailChangeRequest[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const query = pageQuery({ limit: MAX_PAGE_SIZE, offset }, { status: "pending" });
    const response = await apiFetch(`/admin/email-change-requests${query}`, {
      sessionToken: token,
    });
    const page = (await response.json()) as ApiSchema<"EmailChangeRequestPage">;
    requests.push(...page.items.map(fromApiEmailChange));
    if (page.items.length === 0 || requests.length >= page.total) return requests;
  }
}

/**
 * Accepte ou refuse un changement d'adresse email (identifiant de la demande).
 * API : `POST /admin/email-change-requests/{id}/approve`, ou `…/reject`.
 */
export async function reviewEmailChange(
  token: string,
  id: string,
  approve: boolean,
): Promise<EmailChangeReviewResult> {
  if (!isApiEnabled) return reviewEmailChangeWithMocks(token, id, approve);

  try {
    await apiFetch(
      `/admin/email-change-requests/${encodeURIComponent(id)}/${approve ? "approve" : "reject"}`,
      { method: "POST", sessionToken: token },
    );
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.status === 404) return { ok: false, error: "not-found" };
      if (error.code === "email_change_not_pending")
        return { ok: false, error: "already-reviewed" };
      if (error.code === "email_already_used") return { ok: false, error: "email-already-used" };
    }
    return toApiFailure(error, "Décision sur un changement d'email");
  }
}

async function reviewEmailChangeWithMocks(
  token: string,
  id: string,
  approve: boolean,
): Promise<EmailChangeReviewResult> {
  await userFromToken(token, "admin");
  const account = citizensMock.find((existing) => existing.id === id);
  if (!account?.pendingEmail) return mockResponse({ ok: false, error: "not-found" });

  if (approve) {
    const login = accountsMock.find(({ user }) => user.id === id);
    if (login) login.email = account.pendingEmail;
    account.email = account.pendingEmail;
  }
  account.pendingEmail = null;
  return mockResponse({ ok: true });
}
