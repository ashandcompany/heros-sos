/** État renvoyé par une Server Action de formulaire et consommé par `useActionState`. */
export interface FormState<Field extends string = string> {
  status: "idle" | "success" | "error";
  /** Message global (confirmation ou erreur non liée à un champ). */
  message?: string;
  fieldErrors: Partial<Record<Field, string>>;
  /** Valeurs renvoyées pour pré-remplir le formulaire après une erreur. */
  values?: Partial<Record<Field, string>>;
}

export const idleFormState: FormState = { status: "idle", fieldErrors: {} };

export function hasFieldErrors(errors: Partial<Record<string, string | undefined>>): boolean {
  return Object.values(errors).some(Boolean);
}
