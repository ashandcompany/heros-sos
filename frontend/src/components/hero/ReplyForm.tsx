"use client";

import { useActionState } from "react";
import { replyToReviewAction } from "@/app/actions/hero";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { idleFormState, type FormState } from "@/lib/validation/form-state";
import { REPLY_MAX_LENGTH } from "@/lib/validation/hero";

export function ReplyForm({ reviewId }: { reviewId: string }) {
  const [state, formAction] = useActionState(
    replyToReviewAction,
    idleFormState as FormState<"reply">,
  );

  return (
    <form action={formAction} noValidate className="space-y-3">
      <input type="hidden" name="reviewId" value={reviewId} />
      <TextAreaField
        label="Répondre au commentaire"
        hint="Votre réponse sera visible publiquement sous le commentaire."
        name="reply"
        rows={3}
        required
        maxLength={REPLY_MAX_LENGTH}
        defaultValue={state.values?.reply}
        error={state.fieldErrors.reply}
      />
      <FormMessage state={state} />
      <SubmitButton variant="secondary" pendingLabel="Envoi…">
        Publier ma réponse
      </SubmitButton>
    </form>
  );
}
