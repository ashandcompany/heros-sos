import type {
  MfaEnableResult,
  MfaOverview,
  MfaRecoveryCodesResult,
  MfaSession,
  MfaSetupResult,
  MfaVerifyInput,
  MfaVerifyResult,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import {
  ApiError,
  type ApiSchema,
  apiFetch,
  fromApiMfaStatus,
  fromApiUser,
  readSessionCookie,
  toApiFailure,
} from "./http";
import { userFromToken } from "./session-user";

/*
 * Double authentification (TOTP) : obligatoire pour les modérateurs, facultative pour les citoyens et les
 * super-héros. Activation, validation du code à la connexion, codes de secours. Une session limitée (code à saisir, ou activation obligatoire pas encore faite) peut appeler
 * `setup`, `enable` et `verify` ; les codes de secours exigent une session complète.
 *
 * Sans API, aucune 2FA n'existe : ces fonctions simulent l'activation pour que l'écran reste utilisable.
 */

const MOCK_SECRET = "JBSWY3DPEHPK3PXPJBSWY3DPEHPK3PXP";
const MOCK_RECOVERY_CODES = ["ABCD-EFGH-IJKL-MNOP", "QRST-UVWX-YZ23-4567"];
const MOCK_INVALID_CODE = "000000";

/**
 * État de la 2FA de la session et nombre de codes de secours restants.
 * API : `GET /auth/mfa`.
 */
export async function getMfaOverview(token: string): Promise<MfaOverview> {
  if (!isApiEnabled) return mockResponse({ status: "not-enrolled", recoveryCodesRemaining: 0 });

  const response = await apiFetch("/auth/mfa", { sessionToken: token });
  const body = (await response.json()) as ApiSchema<"MfaStatusResponse">;
  return {
    status: fromApiMfaStatus(body.status),
    recoveryCodesRemaining: body.recoveryCodesRemaining,
  };
}

/**
 * Commence l'activation : un secret à configurer dans l'application d'authentification. Recommencer
 * remplace le secret d'une activation non confirmée ; une 2FA déjà activée ne se réinitialise pas.
 * API : `POST /auth/mfa/setup`.
 */
export async function startMfaSetup(token: string): Promise<MfaSetupResult> {
  if (!isApiEnabled) {
    return mockResponse({
      ok: true,
      secret: MOCK_SECRET,
      otpauthUrl: `otpauth://totp/Heros-SOS:demo?secret=${MOCK_SECRET}&issuer=Heros-SOS`,
    });
  }

  try {
    const response = await apiFetch("/auth/mfa/setup", { method: "POST", sessionToken: token });
    const body = (await response.json()) as ApiSchema<"MfaSetupResponse">;
    return { ok: true, secret: body.secret, otpauthUrl: body.otpauthUrl };
  } catch (error) {
    if (error instanceof ApiError && error.code === "mfa_already_enabled") {
      return { ok: false, error: "already-enabled" };
    }
    return toApiFailure(error, "Activation de la double authentification");
  }
}

/** Session complète d'une réponse de l'API : son jeton (cookie `Set-Cookie`) remplace l'ancien. */
function fromApiSession(
  response: Response,
  body: ApiSchema<"LoginResponse"> | ApiSchema<"MfaEnabledResponse">,
): MfaSession {
  return {
    user: fromApiUser(body.user),
    token: readSessionCookie(response),
    expiresAt: body.expiresAt,
  };
}

/**
 * Confirme l'activation avec un premier code. L'API remet les codes de secours (montrés cette seule fois)
 * et ouvre une session complète avec un NOUVEAU jeton : les autres sessions du compte sont fermées.
 * API : `POST /auth/mfa/enable` avec `{ code }`.
 */
export async function enableMfa(token: string, code: string): Promise<MfaEnableResult> {
  if (!isApiEnabled) {
    if (code === MOCK_INVALID_CODE) return mockResponse({ ok: false, error: "invalid-code" });
    const user = await userFromToken(token, ["admin", "citoyen", "super-heros"]);
    return mockResponse({
      ok: true,
      session: { user, token, expiresAt: new Date(Date.now() + 8 * 3_600_000).toISOString() },
      recoveryCodes: MOCK_RECOVERY_CODES,
    });
  }

  try {
    const response = await apiFetch("/auth/mfa/enable", {
      method: "POST",
      sessionToken: token,
      body: { code } satisfies ApiSchema<"MfaCodeRequest">,
    });
    const body = (await response.json()) as ApiSchema<"MfaEnabledResponse">;
    return {
      ok: true,
      session: fromApiSession(response, body),
      recoveryCodes: body.recoveryCodes,
    };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "validation_failed") return { ok: false, error: "invalid-code" };
      if (error.code === "mfa_not_set_up") return { ok: false, error: "not-set-up" };
      if (error.code === "mfa_already_enabled") return { ok: false, error: "already-enabled" };
    }
    return toApiFailure(error, "Confirmation de la double authentification");
  }
}

/**
 * Valide la session d'un compte qui a activé la 2FA : un code de l'application, ou un code de secours (à
 * usage unique). L'API ouvre une session complète avec un NOUVEAU jeton. Un code faux compte comme un
 * échec de connexion : après quelques-uns le compte est verrouillé (`too-many-attempts`).
 * API : `POST /auth/mfa/verify` avec `{ code }` ou `{ recoveryCode }`.
 */
export async function verifyMfa(token: string, input: MfaVerifyInput): Promise<MfaVerifyResult> {
  if (!isApiEnabled) return mockResponse({ ok: false, error: "not-enrolled" });

  try {
    const response = await apiFetch("/auth/mfa/verify", {
      method: "POST",
      sessionToken: token,
      body: input satisfies ApiSchema<"MfaVerifyRequest">,
    });
    const body = (await response.json()) as ApiSchema<"LoginResponse">;
    return { ok: true, session: fromApiSession(response, body) };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "validation_failed") return { ok: false, error: "invalid-code" };
      if (error.code === "mfa_not_set_up") return { ok: false, error: "not-enrolled" };
    }
    return toApiFailure(error, "Validation de la double authentification");
  }
}

/**
 * Remplace les codes de secours (épuisés, ou perdus) : l'API exige le code TOTP courant, une session volée
 * ne suffit pas à s'en procurer. Ils sont montrés cette seule fois.
 * API : `POST /auth/mfa/recovery-codes` avec `{ code }`.
 */
export async function regenerateMfaRecoveryCodes(
  token: string,
  code: string,
): Promise<MfaRecoveryCodesResult> {
  if (!isApiEnabled) {
    if (code === MOCK_INVALID_CODE) return mockResponse({ ok: false, error: "invalid-code" });
    return mockResponse({ ok: true, recoveryCodes: MOCK_RECOVERY_CODES });
  }

  try {
    const response = await apiFetch("/auth/mfa/recovery-codes", {
      method: "POST",
      sessionToken: token,
      body: { code } satisfies ApiSchema<"MfaCodeRequest">,
    });
    const body = (await response.json()) as ApiSchema<"RecoveryCodesResponse">;
    return { ok: true, recoveryCodes: body.recoveryCodes };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "validation_failed") return { ok: false, error: "invalid-code" };
      if (error.code === "mfa_not_set_up") return { ok: false, error: "not-enrolled" };
    }
    return toApiFailure(error, "Codes de secours de la double authentification");
  }
}
