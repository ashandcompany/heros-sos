import { citizensMock } from "@/mocks/citizens";
import type {
  AccountFilters,
  AccountModerationAction,
  AccountModerationResult,
  AccountStatus,
  AccountSummary,
  CitizenAccount,
  Page,
  PageRequest,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, fromApiRole, roleToApi, toApiFailure } from "./http";
import { fromApiPage, pageQuery, toMockPage } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Modération des comptes (back-office) : liste de tous les comptes, désactivation (connexion
 * impossible, sessions fermées), réactivation, et vérification d'un citoyen (information de
 * confiance, sans droit supplémentaire).
 */

type ApiAccountStatus = ApiSchema<"UserSummary">["status"];

/** État d'un compte de l'API vers celui du frontend (aussi lu par les fiches de super-héros). */
export const accountStatusFromApi: Record<ApiAccountStatus, AccountStatus> = {
  pending_activation: "en-attente-activation",
  active: "actif",
  deactivated: "desactive",
};

const statusToApi: Record<AccountStatus, ApiAccountStatus> = {
  "en-attente-activation": "pending_activation",
  actif: "active",
  desactive: "deactivated",
};

function fromApiUserSummary(user: ApiSchema<"UserSummary">): AccountSummary {
  return {
    id: user.id,
    email: user.email,
    firstName: user.firstName,
    lastName: user.lastName,
    alias: user.alias ?? null,
    role: fromApiRole(user.role),
    status: accountStatusFromApi[user.status],
    verifiedAt: user.verifiedAt ?? null,
    registeredAt: user.createdAt,
  };
}

/**
 * Comptes, filtrés par rôle, état ou début d'email.
 * API : `GET /admin/users?role=…&status=…&email=…&limit=…&offset=…`.
 */
export async function getAccounts(
  token: string,
  { role, status, email, ...page }: AccountFilters & PageRequest = {},
): Promise<Page<AccountSummary>> {
  if (!isApiEnabled) return getAccountsWithMocks(token, { role, status, email, ...page });

  const query = pageQuery(page, {
    role: role && roleToApi(role),
    status: status && statusToApi[status],
    email: email?.trim(),
  });
  const response = await apiFetch(`/admin/users${query}`, { sessionToken: token });
  return fromApiPage((await response.json()) as ApiSchema<"UserSummaryPage">, fromApiUserSummary);
}

const moderationErrorByCode = new Map([
  ["not_found", "not-found"],
  ["self_action_forbidden", "self-action"],
  ["protected_account", "protected-account"],
] as const);

/**
 * Désactive, réactive ou vérifie un compte. Désactiver ferme aussitôt toutes ses sessions ;
 * impossible sur son propre compte ou sur un compte de modérateur.
 * API : `POST /admin/users/{id}/deactivate`, `…/reactivate` ou `…/verify`.
 */
export async function moderateAccount(
  token: string,
  id: string,
  action: AccountModerationAction,
): Promise<AccountModerationResult> {
  if (!isApiEnabled) return moderateAccountWithMocks(token, id, action);

  try {
    await apiFetch(`/admin/users/${encodeURIComponent(id)}/${action}`, {
      method: "POST",
      sessionToken: token,
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      const known = moderationErrorByCode.get(error.code as never);
      if (known) return { ok: false, error: known };
    }
    return toApiFailure(error, "Modération d'un compte");
  }
}

/* ---------- Mocks ---------- */

/*
 * Les comptes mock sont les citoyens de `citizensMock`. Leur champ `moderation` sert encore au
 * mock de connexion (`rejete` : compte suspendu) : il reste aligné sur l'état ci-dessous.
 */

/** Vérifications des comptes mock (compte → date), initialisées avec les comptes validés. */
const mockVerifiedAt = new Map(
  citizensMock
    .filter(({ moderation }) => moderation === "valide")
    .map(({ id, registeredAt }) => [id, registeredAt]),
);

function toMockSummary(account: CitizenAccount): AccountSummary {
  return {
    id: account.id,
    email: account.email,
    firstName: account.firstName,
    lastName: account.lastName,
    alias: null,
    role: "citoyen",
    status: account.moderation === "rejete" ? "desactive" : "actif",
    verifiedAt: mockVerifiedAt.get(account.id) ?? null,
    registeredAt: account.registeredAt,
  };
}

async function getAccountsWithMocks(
  token: string,
  { role, status, email, ...page }: AccountFilters & PageRequest,
): Promise<Page<AccountSummary>> {
  await userFromToken(token, "admin");
  const prefix = email?.trim().toLowerCase();
  const accounts = citizensMock
    .map(toMockSummary)
    .filter(
      (account) =>
        (!role || account.role === role) &&
        (!status || account.status === status) &&
        (!prefix || account.email.toLowerCase().startsWith(prefix)),
    )
    .sort((a, b) => Date.parse(b.registeredAt) - Date.parse(a.registeredAt));
  return mockResponse(toMockPage(accounts, page));
}

async function moderateAccountWithMocks(
  token: string,
  id: string,
  action: AccountModerationAction,
): Promise<AccountModerationResult> {
  const moderator = await userFromToken(token, "admin");
  const account = citizensMock.find((existing) => existing.id === id);
  if (!account) return mockResponse({ ok: false, error: "not-found" });
  if (account.id === moderator.id) return mockResponse({ ok: false, error: "self-action" });

  switch (action) {
    case "deactivate":
      account.moderation = "rejete";
      break;
    case "reactivate":
      account.moderation = mockVerifiedAt.has(id) ? "valide" : "en-attente";
      break;
    case "verify":
      if (!mockVerifiedAt.has(id)) mockVerifiedAt.set(id, new Date().toISOString());
      if (account.moderation === "en-attente") account.moderation = "valide";
      break;
  }
  return mockResponse({ ok: true });
}
