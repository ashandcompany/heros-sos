import type { ContactMessage, ContactMessageResult, ContactSubject } from "@/types";
import { apiFailureMessage, apiFieldErrors, hasFieldErrors, type FormState } from "./form-state";

export const contactFields = ["name", "email", "subject", "message"] as const;
export type ContactField = (typeof contactFields)[number];
export type ContactValues = Record<ContactField, string>;

export const contactSubjectLabels: Record<ContactSubject, string> = {
  question: "Question sur le service",
  compte: "Problème avec mon compte",
  donnees: "Mes données personnelles",
  partenariat: "Presse ou partenariat",
  autre: "Autre demande",
};

export const NAME_MAX_LENGTH = 80;
export const MESSAGE_MIN_LENGTH = 20;
export const MESSAGE_MAX_LENGTH = 2000;

/** Champ piège invisible : seul un robot le remplit. */
export const HONEYPOT_FIELD = "website";

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export function readContactValues(formData: FormData): ContactValues {
  return Object.fromEntries(
    contactFields.map((field) => [field, String(formData.get(field) ?? "").trim()]),
  ) as ContactValues;
}

export function validateContact(values: ContactValues): Partial<Record<ContactField, string>> {
  return {
    name: !values.name
      ? "Indiquez votre nom."
      : values.name.length > NAME_MAX_LENGTH
        ? `Le nom ne doit pas dépasser ${NAME_MAX_LENGTH} caractères.`
        : undefined,
    email: EMAIL_PATTERN.test(values.email)
      ? undefined
      : "Veuillez saisir une adresse email valide.",
    subject:
      values.subject in contactSubjectLabels ? undefined : "Choisissez l'objet de votre demande.",
    message: !values.message
      ? "Écrivez votre message."
      : values.message.length < MESSAGE_MIN_LENGTH
        ? `Le message doit contenir au moins ${MESSAGE_MIN_LENGTH} caractères.`
        : values.message.length > MESSAGE_MAX_LENGTH
          ? `Le message ne doit pas dépasser ${MESSAGE_MAX_LENGTH} caractères.`
          : undefined,
  };
}

export function toContactMessage(values: ContactValues): ContactMessage {
  return { ...values, subject: values.subject as ContactSubject };
}

/** État d'erreur du formulaire de contact quand l'API refuse le message. */
export function contactMessageFailureState(
  result: Extract<ContactMessageResult, { ok: false }>,
  values: ContactValues,
): FormState<ContactField> {
  switch (result.error) {
    case "invalid-fields": {
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
    case "delivery-unavailable":
      return {
        status: "error",
        message: "Le message n'a pas pu être envoyé. Veuillez réessayer plus tard.",
        fieldErrors: {},
        values,
      };
    default:
      return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
  }
}
