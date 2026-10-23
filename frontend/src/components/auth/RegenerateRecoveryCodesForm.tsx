"use client";

import { useActionState } from "react";
import { regenerateRecoveryCodesAction, type MfaRecoveryState } from "@/app/actions/mfa";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextField } from "@/components/ui/TextField";
import { idleFormState } from "@/lib/validation/form-state";
import { RecoveryCodesNotice } from "./RecoveryCodesNotice";

/** Remplace les codes de secours (épuisés, ou perdus) : le code courant de l'application est exigé. */
export function RegenerateRecoveryCodesForm() {
  const [state, formAction] = useActionState(
    regenerateRecoveryCodesAction,
    idleFormState as MfaRecoveryState,
  );

  if (state.status === "success" && state.recoveryCodes) {
    return <RecoveryCodesNotice title="Nouveaux codes de secours." codes={state.recoveryCodes} />;
  }

  return (
    <form action={formAction} noValidate className="space-y-4">
      <FormMessage state={state} />
      <TextField
        label="Code à 6 chiffres"
        hint="Il confirme que c'est bien vous. Les anciens codes de secours cesseront de fonctionner."
        name="code"
        inputMode="numeric"
        autoComplete="one-time-code"
        spellCheck={false}
        maxLength={7}
        required
        error={state.fieldErrors.code}
      />
      <SubmitButton variant="secondary" pendingLabel="Génération…">
        Générer de nouveaux codes de secours
      </SubmitButton>
    </form>
  );
}
