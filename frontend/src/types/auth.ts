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

export interface LoginCredentials {
  /** Adresse email, ou pseudo pour un super-héros. */
  identifier: string;
  password: string;
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

export type SignupResult =
  { ok: true; user: AuthUser; token: string } | { ok: false; error: "email-already-used" };
