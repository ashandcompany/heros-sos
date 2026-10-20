"use server";

import { sendContactMessage } from "@/lib/api";
import {
  HONEYPOT_FIELD,
  readContactValues,
  toContactMessage,
  validateContact,
  type ContactField,
} from "@/lib/validation/contact";
import { hasFieldErrors, type FormState } from "@/lib/validation/form-state";

const SUCCESS_MESSAGE =
  "Votre message a bien été transmis au Bureau. Nous vous répondons par email sous 5 jours ouvrés.";

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
  if (!result.ok) {
    return {
      status: "error",
      message: "Le message n'a pas pu être envoyé. Veuillez réessayer plus tard.",
      fieldErrors: {},
      values,
    };
  }

  return { status: "success", message: SUCCESS_MESSAGE, fieldErrors: {} };
}
