"use client";

import { useActionState } from "react";
import { requestHeroStatusAction } from "@/app/actions/account";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import {
  ALIAS_MAX_LENGTH,
  BIO_MAX_LENGTH,
  REQUEST_MESSAGE_MAX_LENGTH,
  type HeroRequestField,
} from "@/lib/validation/hero";

/** Dépôt (ou nouveau dépôt après un refus) d'une demande de statut super-héros. */
export function HeroRequestForm({ previousAlias }: { previousAlias?: string }) {
  const [state, formAction] = useActionState(
    requestHeroStatusAction,
    idleFormState as FormState<HeroRequestField>,
  );

  if (state.status === "success") return <FormMessage state={state} />;

  return (
    <form action={formAction} className="space-y-5" noValidate>
      <TextField
        name="alias"
        label="Pseudo de super-héros"
        hint="Votre nom public. Lettres, chiffres, espaces et tirets."
        required
        maxLength={ALIAS_MAX_LENGTH}
        defaultValue={state.values?.alias ?? previousAlias}
        error={state.fieldErrors.alias}
      />
      <TextAreaField
        name="bio"
        label="Description publique (facultatif)"
        hint="Vos pouvoirs et votre spécialité, affichés sur votre fiche."
        rows={3}
        maxLength={BIO_MAX_LENGTH}
        defaultValue={state.values?.bio}
        error={state.fieldErrors.bio}
      />
      <TextAreaField
        name="requestMessage"
        label="Message au Bureau (facultatif)"
        hint="Vos motivations et vos états de service. Ce message n'est jamais publié."
        rows={4}
        maxLength={REQUEST_MESSAGE_MAX_LENGTH}
        defaultValue={state.values?.requestMessage}
        error={state.fieldErrors.requestMessage}
      />
      <FormMessage state={state} />
      <SubmitButton pendingLabel="Envoi…">Envoyer ma demande</SubmitButton>
    </form>
  );
}
