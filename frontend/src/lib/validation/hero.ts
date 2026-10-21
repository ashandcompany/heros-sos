/** Longueur maximale de la réponse d'un super-héros à un commentaire. */
export const REPLY_MAX_LENGTH = 500;

/* ---------- Demande de statut super-héros ---------- */

/** Mêmes règles que l'API (`validation.Alias`, `MaxBioLength`, `MaxRequestMessageLength`). */
export const ALIAS_MIN_LENGTH = 2;
export const ALIAS_MAX_LENGTH = 30;
export const BIO_MAX_LENGTH = 1000;
export const REQUEST_MESSAGE_MAX_LENGTH = 2000;
export const REJECTION_REASON_MAX_LENGTH = 1000;

// `String.raw` : sinon le gabarit transforme `\p` en `p` et la classe Unicode est perdue.
const ALIAS_PATTERN = new RegExp(
  String.raw`^[\p{L}\p{N} _-]{${ALIAS_MIN_LENGTH},${ALIAS_MAX_LENGTH}}$`,
  "u",
);

export const heroRequestFields = ["alias", "bio", "requestMessage"] as const;
export type HeroRequestField = (typeof heroRequestFields)[number];
export type HeroRequestValues = Record<HeroRequestField, string>;

export function readHeroRequestValues(formData: FormData): HeroRequestValues {
  return Object.fromEntries(
    heroRequestFields.map((field) => [field, String(formData.get(field) ?? "").trim()]),
  ) as HeroRequestValues;
}

const heroRequestMessages = {
  aliasRequired: "Choisissez votre pseudo de super-héros.",
  aliasInvalid: `Le pseudo doit faire de ${ALIAS_MIN_LENGTH} à ${ALIAS_MAX_LENGTH} caractères : lettres, chiffres, espaces, tirets.`,
  bioTooLong: `La description ne doit pas dépasser ${BIO_MAX_LENGTH} caractères.`,
  requestMessageTooLong: `Le message ne doit pas dépasser ${REQUEST_MESSAGE_MAX_LENGTH} caractères.`,
};

export function validateHeroRequest(
  values: HeroRequestValues,
): Partial<Record<HeroRequestField, string>> {
  return {
    alias: !values.alias
      ? heroRequestMessages.aliasRequired
      : ALIAS_PATTERN.test(values.alias)
        ? undefined
        : heroRequestMessages.aliasInvalid,
    bio: values.bio.length > BIO_MAX_LENGTH ? heroRequestMessages.bioTooLong : undefined,
    requestMessage:
      values.requestMessage.length > REQUEST_MESSAGE_MAX_LENGTH
        ? heroRequestMessages.requestMessageTooLong
        : undefined,
  };
}

/** Message d'un champ refusé par l'API (422) : champ → code (`required`, `invalid`, `too_long`). */
export function heroRequestFieldErrors(
  fields: Record<string, string>,
): Partial<Record<HeroRequestField, string>> {
  return {
    alias:
      fields.alias &&
      (fields.alias === "required"
        ? heroRequestMessages.aliasRequired
        : heroRequestMessages.aliasInvalid),
    bio: fields.bio && heroRequestMessages.bioTooLong,
    requestMessage: fields.requestMessage && heroRequestMessages.requestMessageTooLong,
  };
}

/** Motif de refus : obligatoire, il est communiqué au demandeur. */
export function validateRejectionReason(reason: string): string | undefined {
  if (!reason) return "Indiquez le motif du refus : il sera communiqué au demandeur.";
  if (reason.length > REJECTION_REASON_MAX_LENGTH) {
    return `Le motif ne doit pas dépasser ${REJECTION_REASON_MAX_LENGTH} caractères.`;
  }
  return undefined;
}
