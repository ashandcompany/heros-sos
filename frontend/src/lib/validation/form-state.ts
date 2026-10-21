import { formatRetryDelay } from "@/lib/format";
import type { ApiFailure } from "@/types";

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

/** Message affiché pour un échec commun à tous les appels à l'API (voir `toApiFailure`). */
export function apiFailureMessage(failure: ApiFailure): string {
  switch (failure.error) {
    case "forbidden":
      return "Cette action ne vous est pas autorisée. Rechargez la page, puis réessayez.";
    case "payload-too-large":
      return "Le fichier envoyé est trop volumineux.";
    case "unsupported-media-type":
      return "Ce type de fichier n'est pas accepté.";
    case "too-many-attempts":
      return `Trop de demandes. Réessayez dans ${formatRetryDelay(failure.retryAfterSeconds)}.`;
    case "unavailable":
      return "Le service est momentanément indisponible. Réessayez dans quelques instants.";
  }
}

export function hasFieldErrors(errors: Partial<Record<string, string | undefined>>): boolean {
  return Object.values(errors).some(Boolean);
}
