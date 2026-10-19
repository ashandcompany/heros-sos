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
