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

/**
 * État d'une page qui agit sur un lien reçu par mail (`?token=…`) à la demande d'un bouton : confirmation de
 * la newsletter ou d'une nouvelle adresse, désinscription. `invalid-token` : lien falsifié, expiré ou devenu
 * sans objet ; `unavailable` : toute autre erreur.
 */
export type TokenFormState =
  | { status: "idle" }
  | { status: "success" }
  | { status: "error"; error: "invalid-token" | "unavailable" };

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

const apiFieldMessages: Record<string, string> = {
  required: "Ce champ est obligatoire.",
  too_short: "Ce champ est trop court.",
  too_long: "Ce champ est trop long.",
};

/**
 * Erreurs de champ de l'API (422 : champ → code) vers les champs d'un formulaire, ceux de `allowed`
 * seulement ; un champ imbriqué perd son préfixe (`address.city` → `city`). Les règles étant
 * normalement déjà vérifiées avant l'appel, le message reste générique.
 */
export function apiFieldErrors<Field extends string>(
  fields: Record<string, string>,
  allowed: readonly Field[],
): Partial<Record<Field, string>> {
  const errors: Partial<Record<Field, string>> = {};
  for (const [apiField, code] of Object.entries(fields)) {
    const field = apiField.replace(/^address\./, "");
    if ((allowed as readonly string[]).includes(field)) {
      errors[field as Field] = apiFieldMessages[code] ?? "Cette valeur n'est pas valide.";
    }
  }
  return errors;
}

export function hasFieldErrors(errors: Partial<Record<string, string | undefined>>): boolean {
  return Object.values(errors).some(Boolean);
}
