"use client";

import { useActionState } from "react";
import { replyToReviewAction } from "@/app/actions/hero";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import { REPLY_MAX_LENGTH } from "@/lib/validation/hero";

/**
 * Répond au commentaire d'un avis, ou modifie la réponse existante (`currentReply`) : une
 * réponse modifiée repasse en vérification par la modération avant d'être de nouveau publiée.
 */
export function ReplyForm({ reviewId, currentReply }: { reviewId: string; currentReply?: string }) {
  const [state, formAction] = useActionState(
    replyToReviewAction,
    idleFormState as FormState<"reply">,
  );
  const editing = currentReply !== undefined;

  return (
    <form action={formAction} noValidate className="space-y-3">
      <input type="hidden" name="reviewId" value={reviewId} />
      {editing && <input type="hidden" name="currentReply" value={currentReply} />}
      <TextAreaField
        label={editing ? "Nouvelle réponse" : "Répondre au commentaire"}
        hint={
          editing
            ? "Votre réponse modifiée sera masquée le temps que la modération la vérifie."
            : "Votre réponse sera visible publiquement sous le commentaire."
        }
        name="reply"
        rows={3}
        required
        maxLength={REPLY_MAX_LENGTH}
        defaultValue={state.values?.reply ?? currentReply}
        error={state.fieldErrors.reply}
      />
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Envoi…">
        {editing ? "Enregistrer la modification" : "Publier ma réponse"}
      </SubmitButton>
    </form>
  );
}
