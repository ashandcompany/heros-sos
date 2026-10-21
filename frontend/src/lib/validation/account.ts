import type { AvatarUpdateResult, ContactUpdateResult, EmailChangeResult } from "@/types";
import { apiFailureMessage, apiFieldErrors, hasFieldErrors, type FormState } from "./form-state";
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

/** État d'erreur du formulaire de coordonnées quand l'API refuse la modification. */
export function contactFailureState(
  result: Extract<ContactUpdateResult, { ok: false }>,
  values: ContactValues,
): FormState<ContactField> {
  if (result.error !== "invalid-fields") {
    return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
  }
  const fieldErrors = apiFieldErrors(result.fields, contactFields);
  return hasFieldErrors(fieldErrors)
    ? { status: "error", fieldErrors, values }
    : {
        status: "error",
        message: "Certaines informations ne sont pas valides.",
        fieldErrors: {},
        values,
      };
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

/** Message affiché quand l'API refuse la nouvelle photo. */
export function avatarErrorMessage(result: Extract<AvatarUpdateResult, { ok: false }>): string {
  switch (result.error) {
    case "invalid-image":
      return "Cette image n'a pas pu être lue. Choisissez une photo JPEG, PNG ou WebP.";
    case "image-too-large":
      return "L'image est trop grande : 1024 pixels de côté au maximum.";
    default:
      return apiFailureMessage(result);
  }
}

/** État d'erreur du formulaire de changement d'email quand l'API refuse la demande. */
export function emailChangeFailureState(
  result: Extract<EmailChangeResult, { ok: false }>,
  email: string,
): FormState<"email"> {
  const values = { email };
  switch (result.error) {
    case "invalid-email":
      return {
        status: "error",
        fieldErrors: { email: "L'adresse email n'est pas valide." },
        values,
      };
    case "email-already-used":
      return {
        status: "error",
        fieldErrors: { email: "Cette adresse est déjà utilisée par un autre compte." },
        values,
      };
    case "email-unchanged":
      return {
        status: "error",
        fieldErrors: { email: "C'est déjà votre adresse email actuelle." },
        values,
      };
    default:
      return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
  }
}
