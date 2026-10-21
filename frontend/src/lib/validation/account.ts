import type { AvatarUpdateResult, ContactUpdateResult, EmailChangeResult } from "@/types";
import { apiFailureMessage, hasFieldErrors, type FormState } from "./form-state";
import { emptySignupValues, validateSignup } from "./signup";

export const contactFields = [
  "mobilePhone",
  "street",
  "complement",
  "postalCode",
  "city",
  "country",
] as const;
export type ContactField = (typeof contactFields)[number];
export type ContactValues = Record<ContactField, string>;

const COMPLEMENT_MAX_LENGTH = 100;
const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function readContactValues(formData: FormData): ContactValues {
  return Object.fromEntries(
    contactFields.map((field) => [field, String(formData.get(field) ?? "").trim()]),
  ) as ContactValues;
}

/** Mêmes règles qu'à l'inscription, plus le complément d'adresse (facultatif). */
export function validateContact(values: ContactValues): Partial<Record<ContactField, string>> {
  const { complement, ...signupLike } = values;
  return {
    ...validateSignup({ ...emptySignupValues, ...signupLike }, [
      "mobilePhone",
      "street",
      "postalCode",
      "city",
      "country",
    ]),
    complement:
      complement.length > COMPLEMENT_MAX_LENGTH
        ? `Le complément d'adresse ne doit pas dépasser ${COMPLEMENT_MAX_LENGTH} caractères.`
        : undefined,
  };
}

/** Codes de champ de l'API (réponse 422) → message. Les règles sont normalement déjà vérifiées ici. */
const contactMessageByCode: Record<string, string> = {
  required: "Ce champ est obligatoire.",
  too_long: "Ce champ est trop long.",
};

/** Refus de l'API (`mobilePhone`, `address.city`…) → champs du formulaire de coordonnées. */
function contactFieldErrors(fields: Record<string, string>): Partial<Record<ContactField, string>> {
  const errors: Partial<Record<ContactField, string>> = {};
  for (const [apiField, code] of Object.entries(fields)) {
    const field = apiField.replace(/^address\./, "");
    if ((contactFields as readonly string[]).includes(field)) {
      errors[field as ContactField] =
        contactMessageByCode[code] ?? "Cette valeur n'est pas valide.";
    }
  }
  return errors;
}

/** État du formulaire de coordonnées après un refus de l'API. */
export function contactErrorState(
  result: Extract<ContactUpdateResult, { ok: false }>,
  values: ContactValues,
): FormState<ContactField> {
  if (result.error !== "invalid-fields") {
    return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
  }
  const fieldErrors = contactFieldErrors(result.fields);
  return {
    status: "error",
    message: hasFieldErrors(fieldErrors)
      ? undefined
      : "Certaines informations n'ont pas été acceptées.",
    fieldErrors,
    values,
  };
}

export function avatarErrorMessage(result: Extract<AvatarUpdateResult, { ok: false }>): string {
  switch (result.error) {
    case "invalid-image":
      return "Cette image n'a pas pu être lue : choisissez un fichier JPEG, PNG ou WebP.";
    case "image-too-large":
      return "L'image ne doit pas dépasser 1024 pixels de côté.";
    case "payload-too-large":
      return "L'image ne doit pas dépasser 2 Mo.";
    case "unsupported-media-type":
      return "Formats acceptés : JPEG, PNG ou WebP.";
    default:
      return apiFailureMessage(result);
  }
}

export function emailChangeErrorMessage(result: Extract<EmailChangeResult, { ok: false }>): string {
  switch (result.error) {
    case "email-unchanged":
      return "C'est déjà votre adresse email actuelle.";
    case "email-already-used":
      return "Cette adresse est déjà utilisée par un autre compte.";
    case "invalid-email":
      return "L'adresse email n'est pas valide.";
    default:
      return apiFailureMessage(result);
  }
}

export function validateEmail(email: string): string | undefined {
  if (!email) return "Veuillez saisir la nouvelle adresse email.";
  return EMAIL_PATTERN.test(email) ? undefined : "L'adresse email n'est pas valide.";
}

export const AVATAR_MAX_BYTES = 2 * 1024 * 1024;
export const AVATAR_TYPES = ["image/jpeg", "image/png", "image/webp"];

export function validateAvatar(file: File | null): string | undefined {
  if (!file || file.size === 0) return "Veuillez choisir une image.";
  if (!AVATAR_TYPES.includes(file.type)) return "Formats acceptés : JPEG, PNG ou WebP.";
  if (file.size > AVATAR_MAX_BYTES) return "L'image ne doit pas dépasser 2 Mo.";
  return undefined;
}
