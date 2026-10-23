"use client";

import { useActionState } from "react";
import { addVideoAction } from "@/app/actions/hero";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { TextField } from "@/components/ui/TextField";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import {
  VIDEO_DESCRIPTION_MAX_LENGTH,
  VIDEO_TITLE_MAX_LENGTH,
  VIDEO_URL_MAX_LENGTH,
  type FreeVideoField,
} from "@/lib/validation/video";

/**
 * Ajoute une vidéo hors catalogue IMDb (titre, description, lien) : à la vidéothèque du héros
 * connecté par défaut, ou, avec une action du back-office, à celle de `heroId`.
 */
export function FreeVideoForm({
  heroId,
  action = addVideoAction,
}: {
  heroId?: string;
  action?: (previousState: FormState, formData: FormData) => Promise<FormState>;
}) {
  const [state, formAction] = useActionState(action, idleFormState as FormState<FreeVideoField>);
  const errors = state.fieldErrors;
  const value = (field: FreeVideoField) => state.values?.[field] ?? "";

  return (
    <form action={formAction} noValidate className="space-y-4">
      {heroId && <input type="hidden" name="heroId" value={heroId} />}
      <TextField
        label="Titre"
        name="title"
        required
        maxLength={VIDEO_TITLE_MAX_LENGTH}
        defaultValue={value("title")}
        error={errors.title}
      />
      <TextField
        label="Lien de la vidéo"
        name="videoUrl"
        type="url"
        inputMode="url"
        required
        placeholder="https://"
        hint="Adresse complète de la vidéo, chez l'hébergeur de votre choix."
        maxLength={VIDEO_URL_MAX_LENGTH}
        defaultValue={value("videoUrl")}
        error={errors.videoUrl}
      />
      <TextAreaField
        label="Description (facultative)"
        name="description"
        rows={3}
        maxLength={VIDEO_DESCRIPTION_MAX_LENGTH}
        defaultValue={value("description")}
        error={errors.description}
      />
      <SubmitButton variant="secondary" pendingLabel="Ajout…">
        Ajouter la vidéo
      </SubmitButton>
      <FormMessage state={state} />
    </form>
  );
}
