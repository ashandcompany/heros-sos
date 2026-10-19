"use client";

import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import { finishInterventionAction, setPriorityAction, takeChargeAction } from "@/app/actions/hero";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { SelectField } from "@/components/ui/SelectField";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { incidentPriorityLabels } from "@/lib/format";
import { idleFormState } from "@/lib/validation/form-state";
import type { IncidentPriority } from "@/types";

/* Formulaires de la page de détail d'un incident : ils fonctionnent aussi sans JavaScript. */

export function PriorityForm({
  incidentId,
  priority,
}: {
  incidentId: string;
  priority: IncidentPriority;
}) {
  const [state, formAction] = useActionState(setPriorityAction, idleFormState);

  return (
    <form action={formAction} className="space-y-3">
      <input type="hidden" name="id" value={incidentId} />
      <SelectField label="Priorité" name="priority" defaultValue={priority}>
        {Object.entries(incidentPriorityLabels).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </SelectField>
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Enregistrement…">
        Définir la priorité
      </SubmitButton>
    </form>
  );
}

export function TakeChargeForm({ incidentId }: { incidentId: string }) {
  const [state, formAction] = useActionState(takeChargeAction, idleFormState);

  return (
    <form action={formAction} className="space-y-3">
      <input type="hidden" name="id" value={incidentId} />
      <SubmitButton pendingLabel="Prise en charge…" className="w-full">
        Prendre en charge
      </SubmitButton>
      <FormMessage state={state} />
    </form>
  );
}

function OutcomeButtons() {
  const { pending } = useFormStatus();

  return (
    <div className="flex flex-wrap gap-2">
      <button
        type="submit"
        name="outcome"
        value="reussi"
        disabled={pending}
        className={cn(buttonClasses("primary"), "flex-1 disabled:opacity-60")}
      >
        Réussi
      </button>
      <button
        type="submit"
        name="outcome"
        value="echoue"
        disabled={pending}
        className={cn(buttonClasses("danger"), "flex-1 disabled:opacity-60")}
      >
        Échoué
      </button>
    </div>
  );
}

/** Un bouton d'envoi par résultat : la valeur cliquée est transmise dans `outcome`. */
export function FinishInterventionForm({ incidentId }: { incidentId: string }) {
  const [state, formAction] = useActionState(finishInterventionAction, idleFormState);

  return (
    <form action={formAction} className="space-y-3">
      <input type="hidden" name="id" value={incidentId} />
      <fieldset>
        <legend className="mb-2 text-sm font-medium">Résultat de l&apos;intervention</legend>
        <OutcomeButtons />
      </fieldset>
      <FormMessage state={state} />
    </form>
  );
}
