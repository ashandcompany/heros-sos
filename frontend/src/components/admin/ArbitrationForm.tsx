"use client";

import { useActionState } from "react";
import { arbitrateInterventionAction } from "@/app/actions/admin";
import { ChoiceButtons, InterventionRefInputs } from "@/components/hero/IncidentActions";
import { FormMessage } from "@/components/ui/FormMessage";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { idleFormState } from "@/lib/validation/form-state";
import { OUTCOME_NOTES_MAX_LENGTH } from "@/lib/validation/hero";
import type { InterventionRef } from "@/types";

/**
 * Arbitrage d'un désaccord dans une équipe : la modération tranche le résultat et clôt
 * l'intervention. L'API refuse (message affiché) s'il n'y a pas eu de désaccord.
 */
export function ArbitrationForm({ interventionRef }: { interventionRef: InterventionRef }) {
  const [state, formAction] = useActionState(arbitrateInterventionAction, idleFormState);

  return (
    <form action={formAction} className="space-y-3">
      <InterventionRefInputs interventionRef={interventionRef} />
      <TextAreaField
        label="Motif de la décision (facultatif)"
        name="notes"
        rows={3}
        maxLength={OUTCOME_NOTES_MAX_LENGTH}
      />
      <fieldset>
        <legend className="mb-2 text-sm font-medium">Résultat retenu</legend>
        <ChoiceButtons
          name="outcome"
          choices={[
            { value: "reussi", label: "Réussi", variant: "primary" },
            { value: "echoue", label: "Échoué", variant: "danger" },
          ]}
        />
      </fieldset>
      <FormMessage state={state} />
    </form>
  );
}
