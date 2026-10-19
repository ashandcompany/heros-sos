"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { register } from "@/lib/api";
import { SESSION_COOKIE, sessionCookieOptions } from "@/lib/cookies";
import { routes } from "@/lib/routes";
import {
  firstInvalidField,
  hasErrors,
  readSignupValues,
  stepOf,
  toSignupPayload,
  validateSignup,
  type SignupErrors,
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

  const result = await register(toSignupPayload(values));

  if (!result.ok) {
    return {
      step: 1,
      values: safeValues,
      fieldErrors: { email: "Un compte existe déjà avec cette adresse email." },
    };
  }

  (await cookies()).set(SESSION_COOKIE, result.token, sessionCookieOptions);

  redirect(routes.home);
}
