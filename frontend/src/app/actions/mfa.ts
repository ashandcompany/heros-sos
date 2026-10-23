"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { enableMfa, regenerateMfaRecoveryCodes, startMfaSetup, verifyMfa } from "@/lib/api";
import { SESSION_COOKIE } from "@/lib/cookies";
import { appleOtpauthUrl } from "@/lib/otpauth";
import { accountHomeByRole, routes } from "@/lib/routes";
import { setSessionCookie } from "@/lib/server-session";
import { getSiteDomain } from "@/lib/site";
import { apiFailureMessage, type FormState } from "@/lib/validation/form-state";

/*
 * Server Actions de la double authentification. Elles s'exécutent pour une session LIMITÉE (code à saisir,
 * activation obligatoire) autant que complète : elles lisent donc le cookie elle-mêmes plutôt que de passer
 * par `requireRole`, qui traite une session limitée comme un visiteur.
 */

async function sessionToken(): Promise<string> {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  if (!token) redirect(routes.login);
  return token;
}

const TOTP_PATTERN = /^\d{6}$/;
const INVALID_CODE =
  "Ce code n'est pas valide. Il change toutes les 30 secondes : saisissez celui qui s'affiche maintenant.";

/** Retire les espaces d'un code saisi par groupes (« 123 456 »). */
const readCode = (formData: FormData, name: string) =>
  String(formData.get(name) ?? "").replace(/\s/g, "");

/**
 * Valide la session avec un code de l'application, ou un code de secours : le formulaire n'envoie que l'un
 * des deux champs (`code` ou `recoveryCode`), selon ce que la personne a choisi. Le jeton est remplacé.
 */
export async function verifyMfaAction(
  _previousState: FormState<"code" | "recoveryCode">,
  formData: FormData,
): Promise<FormState<"code" | "recoveryCode">> {
  const token = await sessionToken();
  const useRecoveryCode = formData.has("recoveryCode");
  const field = useRecoveryCode ? "recoveryCode" : "code";
  const recoveryCode = String(formData.get("recoveryCode") ?? "").trim();
  const code = readCode(formData, "code");

  if (useRecoveryCode ? !recoveryCode : !TOTP_PATTERN.test(code)) {
    return {
      status: "error",
      fieldErrors: {
        [field]: useRecoveryCode
          ? "Saisissez l'un de vos codes de secours."
          : "Saisissez le code à 6 chiffres de votre application.",
      },
    };
  }

  const result = await verifyMfa(token, useRecoveryCode ? { recoveryCode } : { code });
  if (result.ok) {
    await setSessionCookie(result.session.token, result.session.expiresAt);
    redirect(accountHomeByRole[result.session.user.role]);
  }

  switch (result.error) {
    case "invalid-code":
      return {
        status: "error",
        fieldErrors: {
          [field]: useRecoveryCode
            ? "Ce code de secours n'est pas valide, ou il a déjà servi."
            : INVALID_CODE,
        },
      };
    case "not-enrolled":
      return {
        status: "error",
        message: "La double authentification n'est pas activée sur ce compte.",
        fieldErrors: {},
      };
    default:
      return { status: "error", message: apiFailureMessage(result), fieldErrors: {} };
  }
}

export type MfaSetupState =
  | { status: "idle" }
  | {
      status: "ready";
      secret: string;
      otpauthUrl: string;
      /** Le même lien, pour l'app Mots de passe d'Apple (voir `appleOtpauthUrl`). */
      appleOtpauthUrl: string;
    }
  | { status: "error"; message: string };

/** Commence l'activation : un secret à saisir dans l'application d'authentification. */
export async function startMfaSetupAction(): Promise<MfaSetupState> {
  const result = await startMfaSetup(await sessionToken());
  if (result.ok) {
    return {
      status: "ready",
      secret: result.secret,
      otpauthUrl: result.otpauthUrl,
      appleOtpauthUrl: appleOtpauthUrl(result.otpauthUrl, getSiteDomain()),
    };
  }

  return {
    status: "error",
    message:
      result.error === "already-enabled"
        ? "La double authentification est déjà activée sur ce compte."
        : apiFailureMessage(result),
  };
}

/** Résultat d'un formulaire qui rend des codes de secours : montrés cette seule fois. */
export interface MfaRecoveryState extends FormState<"code"> {
  recoveryCodes?: string[];
}

/**
 * Confirme l'activation avec un premier code. Le jeton de session est remplacé (l'API ferme toutes les
 * autres sessions du compte) et les codes de secours sont rendus pour être montrés cette seule fois.
 */
export async function enableMfaAction(
  _previousState: MfaRecoveryState,
  formData: FormData,
): Promise<MfaRecoveryState> {
  const token = await sessionToken();
  const code = readCode(formData, "code");
  if (!TOTP_PATTERN.test(code)) {
    return {
      status: "error",
      fieldErrors: { code: "Saisissez le code à 6 chiffres de votre application." },
    };
  }

  const result = await enableMfa(token, code);
  if (result.ok) {
    await setSessionCookie(result.session.token, result.session.expiresAt);
    return { status: "success", fieldErrors: {}, recoveryCodes: result.recoveryCodes };
  }

  switch (result.error) {
    case "invalid-code":
      return { status: "error", fieldErrors: { code: INVALID_CODE } };
    case "not-set-up":
      return {
        status: "error",
        message: "L'activation n'a pas été commencée : relancez-la depuis le début.",
        fieldErrors: {},
      };
    case "already-enabled":
      return {
        status: "error",
        message: "La double authentification est déjà activée sur ce compte.",
        fieldErrors: {},
      };
    default:
      return { status: "error", message: apiFailureMessage(result), fieldErrors: {} };
  }
}

/** Remplace les codes de secours (contre le code courant de l'application) : montrés cette seule fois. */
export async function regenerateRecoveryCodesAction(
  _previousState: MfaRecoveryState,
  formData: FormData,
): Promise<MfaRecoveryState> {
  const token = await sessionToken();
  const code = readCode(formData, "code");
  if (!TOTP_PATTERN.test(code)) {
    return {
      status: "error",
      fieldErrors: { code: "Saisissez le code à 6 chiffres de votre application." },
    };
  }

  const result = await regenerateMfaRecoveryCodes(token, code);
  if (result.ok) {
    return { status: "success", fieldErrors: {}, recoveryCodes: result.recoveryCodes };
  }
  if (result.error === "invalid-code") {
    return { status: "error", fieldErrors: { code: INVALID_CODE } };
  }
  if (result.error === "not-enrolled") {
    return {
      status: "error",
      message: "La double authentification n'est pas activée sur ce compte.",
      fieldErrors: {},
    };
  }
  return { status: "error", message: apiFailureMessage(result), fieldErrors: {} };
}
