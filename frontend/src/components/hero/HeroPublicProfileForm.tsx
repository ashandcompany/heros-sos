"use client";

import { useActionState } from "react";
import { updateHeroPublicProfileAction } from "@/app/actions/hero";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import {
  ALIAS_MAX_LENGTH,
  BIO_MAX_LENGTH,
  type HeroPublicProfileField,
} from "@/lib/validation/hero";

/** Pseudo et description publics du super-héros connecté, affichés sur sa fiche. */
export function HeroPublicProfileForm({ alias, power }: { alias: string; power: string }) {
  const [state, formAction] = useActionState(
    updateHeroPublicProfileAction,
    idleFormState as FormState<HeroPublicProfileField>,
  );
  const errors = state.fieldErrors;
  // Après une erreur, on réaffiche la saisie ; sinon les valeurs enregistrées.
  const value = (field: HeroPublicProfileField, saved: string) => state.values?.[field] ?? saved;

  return (
    <form action={formAction} noValidate className="space-y-4">
      <TextField
        label="Pseudo"
        name="alias"
        hint="Unique ; il sert aussi d'identifiant de connexion."
        required
        maxLength={ALIAS_MAX_LENGTH}
        defaultValue={value("alias", alias)}
        error={errors.alias}
      />
      <TextAreaField
        label="Description de vos pouvoirs (facultative)"
        name="power"
        maxLength={BIO_MAX_LENGTH}
        defaultValue={value("power", power)}
        error={errors.power}
      />
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Enregistrement…">
        Enregistrer
      </SubmitButton>
    </form>
  );
}
