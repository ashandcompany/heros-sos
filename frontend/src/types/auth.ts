import type { ApiFailure } from "./api";
import type { MfaStatus } from "./mfa";

export type UserRole = "citoyen" | "super-heros" | "admin";

export interface AuthUser {
  id: string;
  /** Pseudo pour un super-héros, prénom pour un citoyen. */
  displayName: string;
  role: UserRole;
}

export type ActivationResult =
  | { ok: true }
  /** Lien inconnu, expiré (24 h) ou déjà utilisé : l'API ne distingue pas ces cas. */
  | { ok: false; error: "invalid-token" | "unavailable" };

export type ResendActivationResult =
  | { ok: true }
  | {
      ok: false;
      error: "invalid-email" | "too-many-attempts" | "unavailable";
      retryAfterSeconds?: number;
    };

export type PasswordResetConfirmResult =
  | { ok: true }
  /**
   * `invalid-token` : lien inconnu, expiré (1 h) ou déjà utilisé. `weak-password` : refusé par
   * l'API sans consommer le jeton, le visiteur peut réessayer avec le même lien.
   */
  | { ok: false; error: "invalid-token" | "weak-password" | "unavailable" };

export type PasswordResetRequestResult =
  | { ok: true }
  | {
      ok: false;
      error: "invalid-email" | "too-many-attempts" | "unavailable";
      retryAfterSeconds?: number;
    };

export interface LoginCredentials {
  /** Adresse email, ou pseudo pour un super-héros. */
  identifier: string;
  password: string;
  /** « Rester connecté » : session de 30 jours au lieu de 7 (sans effet pour un modérateur). */
  rememberMe?: boolean;
}

export type LoginError =
  | "invalid-credentials"
  /** Compte désactivé par la modération. */
  | "account-suspended"
  /** Compte créé mais pas encore activé par le lien reçu par mail. */
  | "account-not-activated"
  /** Compte verrouillé ou trop de tentatives : voir `retryAfterSeconds`. */
  | "too-many-attempts"
  /** API injoignable ou en erreur. */
  | "unavailable";

export type LoginResult =
  | {
      ok: true;
      user: AuthUser;
      /** Jeton de session, à écrire dans le cookie `SESSION_COOKIE`. */
      token: string;
      /** Fin de validité de la session (ISO 8601), qui fixe l'expiration du cookie. */
      expiresAt: string;
      /** Un modérateur doit encore saisir son code (ou activer la 2FA) : sa session est limitée. */
      mfa: MfaStatus;
    }
  | { ok: false; error: LoginError; retryAfterSeconds?: number };

export type Civility = "madame" | "monsieur" | "non-precise";

export interface PostalAddress {
  street: string;
  postalCode: string;
  city: string;
  /** Code pays ISO 3166-1 alpha-2 (ex : "FR"). */
  country: string;
}

/** Données d'inscription, identiques pour un citoyen et un super-héros. */
export interface SignupPayload {
  civility: Civility;
  firstName: string;
  lastName: string;
  /** Date au format ISO `YYYY-MM-DD`. */
  birthDate: string;
  email: string;
  mobilePhone: string;
  address: PostalAddress;
  password: string;
}

/**
 * Le compte est créé en attente d'activation : aucune session n'est ouverte, un mail contenant le
 * lien d'activation est envoyé à l'adresse saisie.
 */
export type SignupResult =
  | { ok: true; user: AuthUser }
  | { ok: false; error: "email-already-used" | "unavailable" }
  | { ok: false; error: "too-many-attempts"; retryAfterSeconds?: number }
  /** Refusé par l'API (422) : champ du `SignupPayload` (`email`, `address.city`…) → code. */
  | { ok: false; error: "invalid-fields"; fields: Record<string, string> };

/** Fermeture de toutes les sessions du compte : un échec est rapporté (voir `logoutAll`). */
export type LogoutAllResult = { ok: true } | ApiFailure;

/**
 * Changement du mot de passe d'un compte connecté : la session courante continue avec un nouveau jeton
 * (`token`, à écrire dans le cookie), les autres sont fermées.
 */
export type PasswordChangeResult =
  | { ok: true; token: string; expiresAt: string }
  | { ok: false; error: "invalid-current-password" | "weak-password" }
  | ApiFailure;
