import { cache } from "react";
import { accountsMock } from "@/mocks/users";
import { citizensMock, moderatorAccountsMock } from "@/mocks/citizens";
import type {
  AvatarUpdateResult,
  CitizenAccount,
  CitizenAddress,
  ContactUpdate,
  ContactUpdateResult,
  EmailChangeConfirmationResult,
  EmailChangeRequest,
  EmailChangeResult,
  EmailChangeReviewResult,
  MutationResult,
  MyAccount,
  NewsletterPreferenceResult,
} from "@/types";
import { apiFailureMessage } from "@/lib/validation/form-state";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Profil, photo, adresse email et newsletter du compte connecté, et décision de la modération sur
 * les changements d'adresse. `/me` vaut pour tous les rôles : un super-héros est un compte avec un
 * bloc `hero` en plus. Ses fonctions (`super-heroes.ts`) réutilisent donc les appels ci-dessous.
 */

/** Mock : compte du citoyen ou du modérateur connecté (celui du super-héros : `super-heroes.ts`). */
async function findMyAccount(token: string): Promise<CitizenAccount> {
  const user = await userFromToken(token, ["citoyen", "admin"]);
  const accounts = user.role === "admin" ? moderatorAccountsMock : citizensMock;
  const account = accounts.find(({ id }) => id === user.id);
  if (!account) throw new Error("404 : compte introuvable");
  return account;
}

/**
 * Profil du compte connecté, quel que soit son rôle (`GET /me`). Mémorisé le temps d'un rendu :
 * la page et la barre de navigation (photo) le lisent chacune.
 */
export const fetchMyAccount = cache(async (token: string): Promise<ApiSchema<"Account">> => {
  const response = await apiFetch("/me", { sessionToken: token });
  return (await response.json()) as ApiSchema<"Account">;
});

function fromApiAddress({ complement, ...address }: ApiSchema<"AccountAddress">): CitizenAddress {
  return { ...address, complement: complement ?? null };
}

/**
 * Compte du citoyen ou du modérateur connecté.
 * API : `GET /me`. Mock : le profil complet.
 */
export async function getMyAccount(token: string): Promise<MyAccount> {
  if (!isApiEnabled) {
    // Sans API, aucun mail n'est envoyé : une demande est confirmée d'emblée, pour que la modération puisse la traiter.
    const account = await findMyAccount(token);
    return mockResponse({ ...account, pendingEmailConfirmed: account.pendingEmail !== null });
  }

  const account = await fetchMyAccount(token);
  return {
    id: account.id,
    civility: account.civility,
    firstName: account.firstName,
    lastName: account.lastName,
    birthDate: account.birthDate,
    email: account.email,
    pendingEmail: account.pendingEmail ?? null,
    pendingEmailConfirmed: account.pendingEmailConfirmed ?? false,
    mobilePhone: account.mobilePhone,
    address: fromApiAddress(account.address),
    avatarUrl: account.avatarUrl ?? null,
    newsletterSubscribed: account.newsletterSubscribed,
    registeredAt: account.createdAt,
  };
}

/**
 * Modifie le téléphone et l'adresse (`PATCH /me`). Pour l'API, un complément absent ou `null`
 * conserve l'actuel : un citoyen qui vide le champ envoie donc une chaîne vide.
 */
export async function updateMyContact(
  token: string,
  update: ContactUpdate,
): Promise<ContactUpdateResult> {
  if (!isApiEnabled) {
    const account = await findMyAccount(token);
    account.mobilePhone = update.mobilePhone;
    account.address = update.address;
    return mockResponse({ ok: true });
  }

  return patchMyContact(token, {
    mobilePhone: update.mobilePhone,
    address: { ...update.address, complement: update.address.complement ?? "" },
  });
}

/** `PATCH /me`, pour un citoyen comme pour un super-héros. */
export async function patchMyContact(
  token: string,
  body: ApiSchema<"ContactUpdateRequest">,
): Promise<ContactUpdateResult> {
  try {
    await apiFetch("/me", { method: "PATCH", sessionToken: token, body });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError && error.code === "validation_failed") {
      return { ok: false, error: "invalid-fields", fields: error.fields };
    }
    return toApiFailure(error, "Modification des coordonnées");
  }
}

/**
 * Remplace la photo de profil.
 * API : `PUT /me/avatar`, le corps est l'image elle-même (voir `putMyAvatar`). Le mock garde
 * l'image en mémoire sous forme de data URL.
 */
export async function updateMyAvatar(token: string, file: File): Promise<AvatarUpdateResult> {
  if (isApiEnabled) return putMyAvatar(token, file);

  const account = await findMyAccount(token);
  const base64 = Buffer.from(await file.arrayBuffer()).toString("base64");
  account.avatarUrl = `data:${file.type};base64,${base64}`;
  return mockResponse({ ok: true });
}

/**
 * `PUT /me/avatar`, pour un citoyen comme pour un super-héros. Le corps est l'image brute (ni JSON,
 * ni multipart), JPEG, PNG ou WebP : l'API la décode, la ré-encode en JPEG et efface ses métadonnées.
 */
export async function putMyAvatar(token: string, file: File): Promise<AvatarUpdateResult> {
  try {
    await apiFetch("/me/avatar", { method: "PUT", sessionToken: token, file });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError && error.code === "validation_failed") {
      // `avatar` : `required` (corps vide), `invalid` (pas une image lisible) ou `too_large`.
      return {
        ok: false,
        error: error.fields.avatar === "too_large" ? "image-too-large" : "invalid-image",
      };
    }
    return toApiFailure(error, "Photo de profil");
  }
}

/** Retire la photo de profil du citoyen ou du modérateur connecté (voir `deleteMyAvatar`). */
export async function removeMyAvatar(token: string): Promise<MutationResult> {
  if (isApiEnabled) return deleteMyAvatar(token);

  const account = await findMyAccount(token);
  account.avatarUrl = null;
  return mockResponse({ ok: true });
}

/** `DELETE /me/avatar`, pour tous les rôles : l'API supprime aussi le fichier. */
export async function deleteMyAvatar(token: string): Promise<MutationResult> {
  try {
    await apiFetch("/me/avatar", { method: "DELETE", sessionToken: token });
    return { ok: true };
  } catch (error) {
    return { ok: false, error: apiFailureMessage(toApiFailure(error, "Retrait de la photo")) };
  }
}

/**
 * Demande de changement d'email : l'adresse n'est remplacée qu'après validation par la modération.
 * API : `POST /me/email-change-requests` avec `{ email }` (voir `postMyEmailChange`).
 */
export async function requestEmailChange(token: string, email: string): Promise<EmailChangeResult> {
  if (isApiEnabled) return postMyEmailChange(token, email);

  const account = await findMyAccount(token);
  const normalized = email.toLowerCase();

  if (normalized === account.email) {
    return mockResponse({ ok: false, error: "email-unchanged" });
  }
  if (accountsMock.some((existing) => existing.email === normalized)) {
    return mockResponse({ ok: false, error: "email-already-used" });
  }

  account.pendingEmail = normalized;
  return mockResponse({ ok: true });
}

/**
 * `POST /me/email-change-requests`, pour un citoyen comme pour un super-héros. Une nouvelle
 * demande remplace la précédente tant qu'elle n'est pas traitée.
 */
export async function postMyEmailChange(token: string, email: string): Promise<EmailChangeResult> {
  try {
    await apiFetch("/me/email-change-requests", {
      method: "POST",
      sessionToken: token,
      body: { email } satisfies ApiSchema<"EmailChangeRequestPayload">,
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      switch (error.code) {
        case "email_already_used":
          return { ok: false, error: "email-already-used" };
        case "email_unchanged":
          return { ok: false, error: "email-unchanged" };
        case "validation_failed":
          return { ok: false, error: "invalid-email" };
      }
    }
    return toApiFailure(error, "Changement d'adresse email");
  }
}

/**
 * Abonne ou désabonne le titulaire du compte, sans mail de confirmation (son adresse a été vérifiée
 * à l'activation). API : `PUT /me/newsletter` avec `{ subscribed }`, idempotent.
 */
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

/**
 * Confirme la nouvelle adresse d'un compte avec le lien reçu à cette adresse : son titulaire y a bien accès.
 * Cela ne change pas l'adresse du compte, qui reste décidée par la modération. Le lien s'ouvre souvent sur
 * un autre appareil que celui de la demande : aucune session n'est exigée. Cliquer deux fois réussit.
 * API : `POST /email-change-requests/confirm` avec `{ token }`. Mock : tout jeton non vide est accepté.
 */
export async function confirmEmailChange(token: string): Promise<EmailChangeConfirmationResult> {
  if (!isApiEnabled) {
    return mockResponse(token ? { ok: true } : { ok: false, error: "invalid-token" });
  }

  try {
    await apiFetch("/email-change-requests/confirm", {
      method: "POST",
      body: { token } satisfies ApiSchema<"TokenRequest">,
    });
    return { ok: true };
  } catch (error) {
    // Lien invalide (`invalid_token`) ou mal formé (`bad_request`) : même réponse pour le visiteur.
    if (error instanceof ApiError && error.status === 400) {
      return { ok: false, error: "invalid-token" };
    }
    return toApiFailure(error, "Confirmation de la nouvelle adresse email");
  }
}

/* ---------- Back-office (rôle admin) ---------- */

function fromApiEmailChange(item: ApiSchema<"EmailChangeRequestItem">): EmailChangeRequest {
  return {
    id: item.id,
    firstName: item.firstName,
    lastName: item.lastName,
    currentEmail: item.currentEmail,
    newEmail: item.newEmail,
    confirmed: item.confirmedAt !== undefined,
  };
}

/**
 * Changements d'adresse email en attente de validation (citoyens et super-héros), les plus anciens
 * d'abord. Mock : l'identifiant d'une demande est celui du citoyen.
 * API : `GET /admin/email-change-requests?status=pending`, page après page (100 au plus par page).
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
                firstName: account.firstName,
                lastName: account.lastName,
                currentEmail: account.email,
                newEmail: account.pendingEmail,
                confirmed: true,
              },
            ]
          : [],
      ),
    );
  }

  const requests: EmailChangeRequest[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(
      `/admin/email-change-requests${pageQuery({ limit: MAX_PAGE_SIZE, offset }, { status: "pending" })}`,
      { sessionToken: token },
    );
    const page = (await response.json()) as ApiSchema<"EmailChangeRequestPage">;
    requests.push(...page.items.map(fromApiEmailChange));
    if (page.items.length === 0 || requests.length >= page.total) return requests;
  }
}

/**
 * Accepte ou refuse le changement d'adresse email d'une demande. Une demande remplacée par son
 * titulaire change d'identifiant : l'ancien répond « introuvable », pour qu'un modérateur ne valide
 * jamais une adresse qu'il n'a pas vue.
 * API : `POST /admin/email-change-requests/{id}/approve` ou `…/reject`.
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
      switch (error.code) {
        case "email_change_not_found":
          return { ok: false, error: "not-found" };
        case "email_change_not_pending":
          return { ok: false, error: "already-reviewed" };
        case "email_change_not_confirmed":
          return { ok: false, error: "not-confirmed" };
        case "email_already_used":
          return { ok: false, error: "email-already-used" };
      }
    }
    return toApiFailure(error, "Décision sur un changement d'adresse email");
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
