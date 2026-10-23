"use client";

import { useActionState } from "react";
import { saveNewsletterAction } from "@/app/actions/admin";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextField } from "@/components/ui/TextField";
import { NEWSLETTER_SUBJECT_MAX_LENGTH, type NewsletterField } from "@/lib/validation/admin";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import type { NewsletterIssue } from "@/types";
import { RichTextEditor } from "./RichTextEditor";

/** Création (sans `issue`) ou modification d'un brouillon de newsletter. */
export function NewsletterForm({ issue }: { issue?: NewsletterIssue }) {
  const [state, formAction] = useActionState(
    saveNewsletterAction,
    idleFormState as FormState<NewsletterField>,
  );
  const errors = state.fieldErrors;

  return (
    <form action={formAction} noValidate className="space-y-6">
      {issue && <input type="hidden" name="id" value={issue.id} />}
      {issue?.recipientUserId && (
        <input type="hidden" name="recipientUserId" value={issue.recipientUserId} />
      )}

      <TextField
        label="Objet du mail"
        name="subject"
        required
        maxLength={NEWSLETTER_SUBJECT_MAX_LENGTH}
        defaultValue={state.values?.subject ?? issue?.subject ?? ""}
        error={errors.subject}
      />

      {/* L'éditeur garde sa saisie après l'envoi (pas de réinitialisation, contrairement aux champs natifs). */}
      <RichTextEditor
        label="Contenu"
        name="body"
        defaultValue={issue?.body ?? ""}
        error={errors.body}
      />

      <p className="text-sm text-muted">
        Le lien de désinscription est ajouté automatiquement en bas de chaque mail.
      </p>

      <FormMessage state={state} />
      <SubmitButton pendingLabel="Enregistrement…">
        {issue ? "Enregistrer le brouillon" : "Créer le brouillon"}
      </SubmitButton>
    </form>
  );
}
