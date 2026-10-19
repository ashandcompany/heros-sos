import type { Civility, PostalAddress } from "./auth";
import type { ModerationStatus } from "./moderation";

export interface CitizenAddress extends PostalAddress {
  /** Complément d'adresse (bâtiment, étage…). */
  complement: string | null;
}

/** Compte d'un citoyen, tel qu'affiché dans « Mon compte ». */
export interface CitizenAccount {
  id: string;
  civility: Civility;
  firstName: string;
  lastName: string;
  /** Date ISO `YYYY-MM-DD`. */
  birthDate: string;
  email: string;
  /** Nouvelle adresse en attente de validation par la modération. */
  pendingEmail: string | null;
  mobilePhone: string;
  address: CitizenAddress;
  /** Photo de profil (URL), `null` pour le placeholder. */
  avatarUrl: string | null;
  newsletterSubscribed: boolean;
  /** Date ISO 8601 de l'inscription. */
  registeredAt: string;
  /** Contrôle du compte par la modération : `rejete` = compte suspendu (connexion impossible). */
  moderation: ModerationStatus;
}

/** Champs modifiables directement par le citoyen (hors photo, email et newsletter). */
export interface ContactUpdate {
  mobilePhone: string;
  address: CitizenAddress;
}

export type MutationResult = { ok: true } | { ok: false; error: string };
