export type UserRole = "citoyen" | "super-heros";

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
  { ok: true; user: AuthUser; token: string } | { ok: false; error: "invalid-credentials" };
