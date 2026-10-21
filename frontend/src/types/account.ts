import type { ApiFailure } from "./api";
import type { Civility, PostalAddress, UserRole } from "./auth";
import type { ModerationStatus } from "./moderation";

export interface CitizenAddress extends PostalAddress {
  /** Complément d'adresse (bâtiment, étage…). */
  complement: string | null;
}

/** « Mon compte » : profil du compte connecté (`GET /me`). */
export interface MyAccount {
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
}

/** Compte d'un citoyen dans les mocks : son profil et son contrôle par la modération. */
export interface CitizenAccount extends MyAccount {
  /** Contrôle du compte par la modération : `rejete` = compte suspendu (connexion impossible). */
  moderation: ModerationStatus;
}

/** Champs modifiables directement par le titulaire du compte (hors photo, email et newsletter). */
export interface ContactUpdate {
  mobilePhone: string;
  /**
   * `complement` : `null` conserve le complément actuel (formulaire sans ce champ, comme celui
   * d'un super-héros), une chaîne vide l'efface.
   */
  address: CitizenAddress;
}

export type ContactUpdateResult =
  | { ok: true }
  /** Refus de l'API (422) : champ (`mobilePhone`, `address.city`…) → code (`invalid`…). */
  | { ok: false; error: "invalid-fields"; fields: Record<string, string> }
  | ApiFailure;

export type AvatarUpdateResult =
  | { ok: true }
  /** Fichier illisible (`invalid`) ou de plus de 1024 pixels de côté (`too_large`). */
  | { ok: false; error: "invalid-image" | "image-too-large" }
  | ApiFailure;

export type EmailChangeResult =
  | { ok: true }
  | { ok: false; error: "email-unchanged" | "email-already-used" | "invalid-email" }
  | ApiFailure;

export type NewsletterPreferenceResult = { ok: true } | ApiFailure;

/** Demande de changement d'adresse email, vue par la modération (`EmailChangeRequestItem`). */
export interface EmailChangeRequest {
  /** Identifiant de la demande : il change quand le titulaire la remplace par une nouvelle. */
  id: string;
  userId: string;
  firstName: string;
  lastName: string;
  currentEmail: string;
  newEmail: string;
  /** Date ISO 8601 de dépôt. */
  submittedAt: string;
}

export type EmailChangeReviewResult =
  | { ok: true }
  | {
      ok: false;
      /**
       * Demande introuvable (ou remplacée par son titulaire), déjà traitée, ou adresse prise
       * entre-temps par un autre compte (la demande est alors à refuser).
       */
      error: "not-found" | "already-reviewed" | "email-already-used";
    }
  | ApiFailure;

export type MutationResult = { ok: true } | { ok: false; error: string };

/** État d'un compte (`UserSummary.status`) : un compte désactivé ne peut plus se connecter. */
export type AccountStatus = "en-attente-activation" | "actif" | "desactive";

/** Compte vu par la modération (schéma `UserSummary`), quel que soit son rôle. */
export interface AccountSummary {
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  /** Pseudo, pour un super-héros. */
  alias: string | null;
  role: UserRole;
  status: AccountStatus;
  /** Date ISO 8601 de vérification par un modérateur (information de confiance), `null` sinon. */
  verifiedAt: string | null;
  /** Date ISO 8601 de création du compte. */
  registeredAt: string;
}

/** Filtres de la liste des comptes (`GET /admin/users`). */
export interface AccountFilters {
  role?: UserRole;
  status?: AccountStatus;
  /** Début de l'adresse email, sans tenir compte de la casse. */
  email?: string;
}

/** Actions de modération sur un compte : routes `/admin/users/{id}/…`. */
export type AccountModerationAction = "deactivate" | "reactivate" | "verify";

export type AccountModerationResult =
  | { ok: true }
  | {
      ok: false;
      /** Compte introuvable, action sur son propre compte, ou compte de modérateur. */
      error: "not-found" | "self-action" | "protected-account";
    }
  | ApiFailure;
