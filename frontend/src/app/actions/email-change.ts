"use server";

import { confirmEmailChange } from "@/lib/api";
import type { TokenFormState } from "@/lib/validation/form-state";

/**
 * Confirme la nouvelle adresse d'un compte avec le lien reçu à cette adresse. Aucune session n'est exigée : le
 * lien s'ouvre souvent sur un autre appareil. L'adresse du compte, elle, ne change qu'après la modération.
 */
export async function confirmEmailChangeAction(
  _previousState: TokenFormState,
  formData: FormData,
): Promise<TokenFormState> {
  const result = await confirmEmailChange(String(formData.get("token") ?? ""));
  if (!result.ok) {
    return {
      status: "error",
      error: result.error === "invalid-token" ? "invalid-token" : "unavailable",
    };
  }
  return { status: "success" };
}
