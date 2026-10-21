import type { ApiFailure } from "./api";
import type { Civility, PostalAddress, UserRole } from "./auth";
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

/**
 * « Mon compte » : le profil du citoyen connecté (`GET /me`). Son contrôle par la modération n'en fait pas
 * partie ; `pendingEmailConfirmed` dit si le titulaire de la nouvelle adresse a cliqué sur le lien reçu.
 */
export type MyAccount = Omit<CitizenAccount, "moderation"> & { pendingEmailConfirmed: boolean };

/** Champs modifiables directement par le citoyen (hors photo, email et newsletter). */
export interface ContactUpdate {
  mobilePhone: string;
  address: CitizenAddress;
}

export type MutationResult = { ok: true } | { ok: false; error: string };

/** Champs refusés par l'API (422) : champ (`mobilePhone`, `address.city`…) → code (`invalid`…). */
type InvalidFields = { ok: false; error: "invalid-fields"; fields: Record<string, string> };

export type ContactUpdateResult = { ok: true } | InvalidFields | ApiFailure;

export type AvatarUpdateResult =
  | { ok: true }
  | {
      ok: false;
      /** Ce n'est pas une image JPEG, PNG ou WebP lisible, ou elle dépasse 1024 pixels de côté. */
      error: "invalid-image" | "image-too-large";
    }
  | ApiFailure;

export type EmailChangeResult =
  | { ok: true }
  | {
      ok: false;
      /** Adresse mal formée, déjà prise par un autre compte, ou déjà celle de ce compte. */
      error: "invalid-email" | "email-already-used" | "email-unchanged";
    }
  | ApiFailure;

export type NewsletterPreferenceResult = { ok: true } | ApiFailure;

/** Demande de changement d'adresse email en attente, vue par la modération (`EmailChangeRequestItem`). */
export interface EmailChangeRequest {
  /** Identifiant de la demande (routes de décision) : il change quand le titulaire la remplace. */
  id: string;
  firstName: string;
  lastName: string;
  currentEmail: string;
  newEmail: string;
  /**
   * Le titulaire de la nouvelle adresse l'a confirmée par le lien reçu. Tant qu'elle ne l'est pas, la demande
   * ne peut pas être acceptée, seulement refusée.
   */
  confirmed: boolean;
}

export type EmailChangeConfirmationResult =
  | { ok: true }
  /** Lien falsifié, expiré, ou devenu sans objet (demande remplacée ou refusée). */
  | { ok: false; error: "invalid-token" }
  | ApiFailure;

export type EmailChangeReviewResult =
  | { ok: true }
  | {
      ok: false;
      /**
       * Demande introuvable (remplacée par son titulaire), déjà traitée, adresse pas encore confirmée par son
       * titulaire, ou prise entre-temps (elle ne peut plus qu'être refusée).
       */
      error: "not-found" | "already-reviewed" | "not-confirmed" | "email-already-used";
    }
  | ApiFailure;

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
