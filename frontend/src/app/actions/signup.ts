"use server";

import { register } from "@/lib/api";
import {
  firstInvalidField,
  hasErrors,
  PASSWORD_RULES,
  readSignupValues,
  signupFields,
  stepOf,
  toSignupPayload,
  validateSignup,
  type SignupErrors,
  type SignupField,
  type SignupStep,
  type SignupValues,
} from "@/lib/validation/signup";

export interface SignupFormState {
  /** Étape à afficher : celle du premier champ en erreur. */
  step: SignupStep;
  /** Valeurs renvoyées pour pré-remplir le formulaire (mots de passe toujours vidés). */
  values: SignupValues;
  fieldErrors: SignupErrors;
  formError?: string;
  /** Compte créé : adresse à laquelle le lien d'activation a été envoyé. */
  registeredEmail?: string;
}

/** Codes de champ de l'API (réponse 422) → message. Les règles sont normalement déjà vérifiées ici. */
const fieldMessageByCode: Record<string, string> = {
  required: "Ce champ est obligatoire.",
  too_short: "Ce champ est trop court.",
  too_long: "Ce champ est trop long.",
  weak: `Le mot de passe est trop faible. ${PASSWORD_RULES}`,
};

/** Erreurs de l'API → champs du formulaire (`address.city` → `city`). */
function toSignupErrors(fields: Record<string, string>): SignupErrors {
  const errors: SignupErrors = {};
  for (const [apiField, code] of Object.entries(fields)) {
    const field = apiField.replace(/^address\./, "");
    if ((signupFields as readonly string[]).includes(field)) {
      errors[field as SignupField] = fieldMessageByCode[code] ?? "Cette valeur n'est pas valide.";
    }
  }
  return errors;
}

export async function signupAction(
  _previousState: SignupFormState,
  formData: FormData,
): Promise<SignupFormState> {
  const values = readSignupValues(formData);
  const safeValues = { ...values, password: "", passwordConfirmation: "" };
  const fieldErrors = validateSignup(values);

  if (hasErrors(fieldErrors)) {
    const field = firstInvalidField(fieldErrors);
    return { step: field ? stepOf(field) : 1, values: safeValues, fieldErrors };
  }

  const payload = toSignupPayload(values);
  const result = await register(payload);

  if (result.ok) {
    return { step: 2, values: safeValues, fieldErrors: {}, registeredEmail: payload.email };
  }

  switch (result.error) {
    case "email-already-used":
      return {
        step: 1,
        values: safeValues,
        fieldErrors: { email: "Un compte existe déjà avec cette adresse email." },
      };
    case "invalid-fields": {
      const apiErrors = toSignupErrors(result.fields);
      const field = firstInvalidField(apiErrors);
      return {
        step: field ? stepOf(field) : 2,
        values: safeValues,
        fieldErrors: apiErrors,
        formError: field ? undefined : "Certaines informations n'ont pas été acceptées.",
      };
    }
    case "too-many-attempts": {
      const minutes = Math.ceil((result.retryAfterSeconds ?? 60) / 60);
      return {
        step: 2,
        values: safeValues,
        fieldErrors: {},
        formError: `Trop d'inscriptions depuis votre connexion. Réessayez dans ${minutes} minute${minutes > 1 ? "s" : ""}.`,
      };
    }
    case "unavailable":
      return {
        step: 2,
        values: safeValues,
        fieldErrors: {},
        formError:
          "L'inscription est momentanément indisponible. Réessayez dans quelques instants.",
      };
  }
}
