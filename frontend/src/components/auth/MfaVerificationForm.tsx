"use client";

import { useActionState, useState } from "react";
import { verifyMfaAction } from "@/app/actions/mfa";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { TextField } from "@/components/ui/TextField";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

/**
 * Saisie du code de l'application d'authentification, ou à défaut d'un code de secours. Le formulaire
 * n'envoie que le champ affiché : l'action en déduit ce qui est utilisé.
 */
export function MfaVerificationForm() {
  const [useRecoveryCode, setUseRecoveryCode] = useState(false);
  const [state, formAction, isPending] = useActionState(
    verifyMfaAction,
    idleFormState as FormState<"code" | "recoveryCode">,
  );

  return (
    <form action={formAction} noValidate className="space-y-5">
      <FormMessage state={state} />

      {useRecoveryCode ? (
        <TextField
          key="recoveryCode"
          label="Code de secours"
          hint="L'un des codes remis à l'activation, avec ou sans tirets. Chaque code ne sert qu'une fois."
          name="recoveryCode"
          autoComplete="off"
          autoCapitalize="characters"
          spellCheck={false}
          required
          error={state.fieldErrors.recoveryCode}
        />
      ) : (
        <TextField
          key="code"
          label="Code à 6 chiffres"
          hint="Celui que votre application d'authentification affiche maintenant."
          name="code"
          inputMode="numeric"
          autoComplete="one-time-code"
          spellCheck={false}
          maxLength={7}
          required
          error={state.fieldErrors.code}
        />
      )}

      <button
        type="submit"
        disabled={isPending}
        className={cn(buttonClasses("primary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? "Vérification…" : "Valider"}
      </button>

      <button
        type="button"
        onClick={() => setUseRecoveryCode((current) => !current)}
        className="w-full rounded-sm text-center text-sm font-medium text-accent hover:underline"
      >
        {useRecoveryCode ? "Utiliser le code de mon application" : "J'ai perdu mon téléphone"}
      </button>
    </form>
  );
}
