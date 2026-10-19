import { countries, DEFAULT_COUNTRY } from "@/lib/countries";
import type { Civility, SignupPayload } from "@/types";

/**
 * Validation du formulaire d'inscription, partagée entre le client (passage d'étape)
 * et la Server Action (validation finale).
 */

export const signupFields = [
  "civility",
  "firstName",
  "lastName",
  "birthDate",
  "email",
  "mobilePhone",
  "street",
  "postalCode",
  "city",
  "country",
  "password",
  "passwordConfirmation",
] as const;

export type SignupField = (typeof signupFields)[number];
export type SignupValues = Record<SignupField, string>;
export type SignupErrors = Partial<Record<SignupField, string>>;
export type SignupStep = 1 | 2;

export const signupSteps: Array<{ step: SignupStep; title: string; fields: SignupField[] }> = [
  {
    step: 1,
    title: "Informations personnelles",
    fields: ["civility", "firstName", "lastName", "birthDate", "email"],
  },
  {
    step: 2,
    title: "Coordonnées et sécurité",
    fields: [
      "mobilePhone",
      "street",
      "postalCode",
      "city",
      "country",
      "password",
      "passwordConfirmation",
    ],
  },
];

export const civilityOptions: Array<{ value: Civility; label: string }> = [
  { value: "madame", label: "Madame" },
  { value: "monsieur", label: "Monsieur" },
  { value: "non-precise", label: "Ne pas préciser" },
];

export const PASSWORD_MIN_LENGTH = 10;
export const PASSWORD_RULES = `Au moins ${PASSWORD_MIN_LENGTH} caractères, dont une majuscule, une minuscule et un chiffre.`;

export const emptySignupValues: SignupValues = {
  ...(Object.fromEntries(signupFields.map((field) => [field, ""])) as SignupValues),
  country: DEFAULT_COUNTRY,
};

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
const NAME_PATTERN = /^\p{L}[\p{L}' -]{0,49}$/u;
const ISO_DATE_PATTERN = /^\d{4}-\d{2}-\d{2}$/;
const FR_MOBILE_PATTERN = /^0[67]\d{8}$/;
const INTERNATIONAL_PHONE_PATTERN = /^\+\d{8,15}$/;
const FR_POSTAL_CODE_PATTERN = /^\d{5}$/;
const POSTAL_CODE_PATTERN = /^[A-Za-z0-9][A-Za-z0-9 -]{1,9}$/;
const MAX_AGE_IN_YEARS = 120;

function validateBirthDate(value: string): string | undefined {
  if (!value) return "Veuillez saisir votre date de naissance.";

  const date = new Date(`${value}T00:00:00`);
  if (!ISO_DATE_PATTERN.test(value) || Number.isNaN(date.getTime())) {
    return "La date de naissance n'est pas valide.";
  }

  const now = new Date();
  const oldest = new Date(now.getFullYear() - MAX_AGE_IN_YEARS, now.getMonth(), now.getDate());
  if (date > now || date < oldest) return "La date de naissance n'est pas valide.";

  return undefined;
}

const validators: Record<SignupField, (value: string, values: SignupValues) => string | undefined> =
  {
    civility: (value) =>
      civilityOptions.some((option) => option.value === value)
        ? undefined
        : "Veuillez choisir une civilité.",
    firstName: (value) => {
      if (!value) return "Veuillez saisir votre prénom.";
      return NAME_PATTERN.test(value) ? undefined : "Le prénom contient des caractères invalides.";
    },
    lastName: (value) => {
      if (!value) return "Veuillez saisir votre nom.";
      return NAME_PATTERN.test(value) ? undefined : "Le nom contient des caractères invalides.";
    },
    birthDate: validateBirthDate,
    email: (value) => {
      if (!value) return "Veuillez saisir votre adresse email.";
      return EMAIL_PATTERN.test(value) ? undefined : "L'adresse email n'est pas valide.";
    },
    mobilePhone: (value) => {
      if (!value) return "Veuillez saisir votre numéro de portable.";
      const phone = value.replace(/[\s.-]/g, "");
      return FR_MOBILE_PATTERN.test(phone) || INTERNATIONAL_PHONE_PATTERN.test(phone)
        ? undefined
        : "Saisissez un numéro de portable valide (ex : 06 12 34 56 78 ou +33 6 12 34 56 78).";
    },
    street: (value) => (value ? undefined : "Veuillez saisir le numéro et le nom de la rue."),
    postalCode: (value, values) => {
      if (!value) return "Veuillez saisir votre code postal.";
      const pattern = values.country === "FR" ? FR_POSTAL_CODE_PATTERN : POSTAL_CODE_PATTERN;
      return pattern.test(value) ? undefined : "Le code postal n'est pas valide.";
    },
    city: (value) => (value ? undefined : "Veuillez saisir votre ville."),
    country: (value) =>
      countries.some((country) => country.code === value) ? undefined : "Veuillez choisir un pays.",
    password: (value) => {
      if (!value) return "Veuillez choisir un mot de passe.";
      const isStrong =
        value.length >= PASSWORD_MIN_LENGTH &&
        /[a-z]/.test(value) &&
        /[A-Z]/.test(value) &&
        /\d/.test(value);
      return isStrong ? undefined : `Le mot de passe est trop faible. ${PASSWORD_RULES}`;
    },
    passwordConfirmation: (value, values) => {
      if (!value) return "Veuillez confirmer votre mot de passe.";
      return value === values.password ? undefined : "Les mots de passe ne correspondent pas.";
    },
  };

export function readSignupValues(formData: FormData): SignupValues {
  const values = { ...emptySignupValues };
  for (const field of signupFields) {
    const raw = String(formData.get(field) ?? "");
    values[field] = field === "password" || field === "passwordConfirmation" ? raw : raw.trim();
  }
  return values;
}

/**
 * Valide les champs demandés. Chaque champ validé apparaît dans le résultat
 * (`undefined` s'il est valide), ce qui permet d'effacer une erreur précédente par fusion.
 */
export function validateSignup(
  values: SignupValues,
  fields: readonly SignupField[] = signupFields,
): SignupErrors {
  return Object.fromEntries(
    fields.map((field) => [field, validators[field](values[field], values)]),
  ) as SignupErrors;
}

export function hasErrors(errors: SignupErrors): boolean {
  return Object.values(errors).some(Boolean);
}

/** Premier champ en erreur, dans l'ordre du formulaire. */
export function firstInvalidField(errors: SignupErrors): SignupField | undefined {
  return signupFields.find((field) => errors[field]);
}

export function stepOf(field: SignupField): SignupStep {
  return signupSteps.find(({ fields }) => fields.includes(field))?.step ?? 1;
}

export function toSignupPayload(values: SignupValues): SignupPayload {
  return {
    civility: values.civility as SignupPayload["civility"],
    firstName: values.firstName,
    lastName: values.lastName,
    birthDate: values.birthDate,
    email: values.email.toLowerCase(),
    mobilePhone: values.mobilePhone.replace(/[\s.-]/g, ""),
    address: {
      street: values.street,
      postalCode: values.postalCode,
      city: values.city,
      country: values.country,
    },
    password: values.password,
  };
}
