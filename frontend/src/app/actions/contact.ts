"use server";

import { sendContactMessage } from "@/lib/api";
import {
  contactFields,
  HONEYPOT_FIELD,
  readContactValues,
  toContactMessage,
  validateContact,
  type ContactField,
} from "@/lib/validation/contact";
import { apiFailureMessage, hasFieldErrors, type FormState } from "@/lib/validation/form-state";

const SUCCESS_MESSAGE =
  "Votre message a bien été transmis au Bureau. Nous vous répondons par email sous 5 jours ouvrés.";

/** Champs refusés par l'API : les règles sont celles de `validateContact`, rarement prises en défaut. */
function toContactErrors(fields: Record<string, string>): Partial<Record<ContactField, string>> {
  const errors: Partial<Record<ContactField, string>> = {};
  for (const field of Object.keys(fields)) {
    if ((contactFields as readonly string[]).includes(field)) {
      errors[field as ContactField] = "Cette valeur n'est pas valide.";
    }
  }
  return errors;
}

export async function sendContactMessageAction(
  _previousState: FormState<ContactField>,
  formData: FormData,
): Promise<FormState<ContactField>> {
  // Robot détecté : on fait comme si tout s'était bien passé, sans rien transmettre.
  if (String(formData.get(HONEYPOT_FIELD) ?? "") !== "") {
    return { status: "success", message: SUCCESS_MESSAGE, fieldErrors: {} };
  }

  const values = readContactValues(formData);
  const fieldErrors = validateContact(values);
  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await sendContactMessage(toContactMessage(values));
  if (result.ok) return { status: "success", message: SUCCESS_MESSAGE, fieldErrors: {} };

  switch (result.error) {
    case "invalid-fields": {
      const apiErrors = toContactErrors(result.fields);
      return {
        status: "error",
        message: hasFieldErrors(apiErrors)
          ? undefined
          : "Certaines informations n'ont pas été acceptées.",
        fieldErrors: apiErrors,
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
