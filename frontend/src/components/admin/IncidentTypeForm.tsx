"use client";

import { useActionState } from "react";
import { createIncidentTypeAction } from "@/app/actions/admin";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextField } from "@/components/ui/TextField";
import {
  INCIDENT_TYPE_CODE_MAX_LENGTH,
  INCIDENT_TYPE_LABEL_MAX_LENGTH,
  type IncidentTypeField,
} from "@/lib/validation/admin";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

/** Ajout d'un type au référentiel. Le formulaire est vidé après une création réussie. */
export function IncidentTypeForm() {
  const [state, formAction] = useActionState(
    createIncidentTypeAction,
    idleFormState as FormState<IncidentTypeField>,
  );
  const values = state.status === "error" ? (state.values ?? {}) : {};

  return (
    // `key` : après une création, le formulaire repart vide.
    <form key={state.message} action={formAction} noValidate className="space-y-5">
      <div className="grid gap-5 sm:grid-cols-2">
        <TextField
          name="label"
          label="Libellé"
          hint="Affiché aux citoyens et aux super-héros."
          required
          maxLength={INCIDENT_TYPE_LABEL_MAX_LENGTH}
          defaultValue={values.label}
          error={state.fieldErrors.label}
        />
        <TextField
          name="code"
          label="Code"
          hint="Identifiant stable : minuscules, chiffres et tirets (ex : prise-d-otages)."
          required
          maxLength={INCIDENT_TYPE_CODE_MAX_LENGTH}
          defaultValue={values.code}
          error={state.fieldErrors.code}
        />
      </div>
      <label className="flex items-center gap-3 text-sm">
        <input
          type="checkbox"
          name="endangersLife"
          defaultChecked={values.endangersLife === "on"}
          className="size-5 accent-primary"
        />
        Ce type d&apos;incident met des vies en danger
      </label>
      <FormMessage state={state} />
      <SubmitButton pendingLabel="Création…">Créer le type</SubmitButton>
    </form>
  );
}
