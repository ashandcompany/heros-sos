import type { ApiFailure } from "./api";
import type { AuthUser } from "./auth";

/**
 * Où en est la double authentification (TOTP) d'une session :
 * - `verified` : activée et validée, la session est complète ;
 * - `verification-required` : activée, le code reste à saisir ; la session est **limitée** ;
 * - `enrollment-required` : obligatoire (modérateur en production) et pas encore activée ; session **limitée** ;
 * - `not-enrolled` : ni activée ni obligatoire, la session est complète.
 *
 * Une session limitée ne sert qu'à la 2FA et à la déconnexion : le frontend la traite comme un visiteur
 * partout ailleurs (voir `getSessionUser`).
 */
export type MfaStatus =
  "verified" | "verification-required" | "enrollment-required" | "not-enrolled";

export interface MfaOverview {
  status: MfaStatus;
  /** Codes de secours qui n'ont pas servi (0 sans 2FA). */
  recoveryCodesRemaining: number;
}

/** Session complète rendue par l'API après l'activation ou la validation : son jeton remplace l'ancien. */
export interface MfaSession {
  user: AuthUser;
  token: string;
  /** Fin de validité de la session (ISO 8601), qui fixe l'expiration du cookie. */
  expiresAt: string;
}

export type MfaSetupResult =
  | {
      ok: true;
      /** Secret en base32, à saisir à la main dans l'application d'authentification. */
      secret: string;
      /** Adresse `otpauth://` qu'une application lit (lien ou QR code). */
      otpauthUrl: string;
    }
  | { ok: false; error: "already-enabled" }
  | ApiFailure;

/** Code faux, périmé ou déjà utilisé : l'API ne distingue pas ces cas. */
type InvalidCode = { ok: false; error: "invalid-code" };

export type MfaEnableResult =
  | { ok: true; session: MfaSession; recoveryCodes: string[] }
  | InvalidCode
  | { ok: false; error: "not-set-up" | "already-enabled" }
  | ApiFailure;

export type MfaVerifyResult =
  | { ok: true; session: MfaSession }
  | InvalidCode
  | { ok: false; error: "not-enrolled" }
  | ApiFailure;

export type MfaRecoveryCodesResult =
  | { ok: true; recoveryCodes: string[] }
  | InvalidCode
  | { ok: false; error: "not-enrolled" }
  | ApiFailure;

/** Ce que saisit l'utilisateur pour valider sa session : un code de l'application, ou un code de secours. */
export type MfaVerifyInput = { code: string } | { recoveryCode: string };
