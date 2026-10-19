export type UserRole = "citoyen" | "super-heros" | "admin";

export interface AuthUser {
  id: string;
  /** Pseudo pour un super-héros, prénom pour un citoyen. */
  displayName: string;
  role: UserRole;
}

export interface LoginCredentials {
  /** Adresse email, ou pseudo pour un super-héros. */
  identifier: string;
  password: string;
}

export type LoginResult =
  | { ok: true; user: AuthUser; token: string }
  | { ok: false; error: "invalid-credentials" | "account-suspended" };

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
