"use client";

import { useActionState } from "react";
import { useFormStatus } from "react-dom";
import { reviewHeroApplicationAction } from "@/app/actions/admin";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { REJECTION_REASON_MAX_LENGTH } from "@/lib/validation/hero";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

function DecisionButtons({ subject }: { subject: string }) {
  const { pending } = useFormStatus();

  return (
    <div className="flex flex-wrap gap-2">
      <button
        type="submit"
        name="decision"
        value="valide"
        disabled={pending}
        className={cn(buttonClasses("primary"), "disabled:opacity-60")}
      >
        Accepter la demande<span className="sr-only"> : {subject}</span>
      </button>
      <button
        type="submit"
        name="decision"
        value="rejete"
        disabled={pending}
        className={cn(buttonClasses("danger"), "disabled:opacity-60")}
      >
        Refuser<span className="sr-only"> : {subject}</span>
      </button>
    </div>
  );
}

/**
 * Décision sur une demande de statut super-héros. Le motif n'est exigé que pour un refus (il est
 * communiqué au demandeur) : l'action le vérifie. Fonctionne sans JavaScript.
 */
export function HeroReviewForm({
  userId,
  subject,
  className,
}: {
  userId: string;
  /** Nom de la demande, ajouté au libellé des boutons pour les lecteurs d'écran. */
  subject: string;
  className?: string;
}) {
  const [state, formAction] = useActionState(
    reviewHeroApplicationAction,
    idleFormState as FormState<"reason">,
  );

  return (
    <form action={formAction} className={cn("space-y-3", className)}>
      <input type="hidden" name="id" value={userId} />
      <TextAreaField
        name="reason"
        label="Motif du refus"
        hint="Obligatoire pour refuser : il est communiqué au demandeur."
        rows={2}
        maxLength={REJECTION_REASON_MAX_LENGTH}
        defaultValue={state.values?.reason}
        error={state.fieldErrors.reason}
      />
      <DecisionButtons subject={subject} />
      <FormMessage state={state} />
    </form>
  );
}
