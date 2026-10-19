"use client";

import { useActionState, useId, useState } from "react";
import { rateInterventionAction } from "@/app/actions/incidents";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextAreaField } from "@/components/ui/TextAreaField";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

const STAR_PATH =
  "M10 1.5l2.6 5.3 5.9.9-4.3 4.1 1 5.8L10 14.9l-5.2 2.7 1-5.8L1.5 7.7l5.9-.9L10 1.5z";

/**
 * Note sur 5 : 5 boutons radio natifs (flèches du clavier, lecteurs d'écran),
 * affichés sous forme d'étoiles.
 */
function StarRatingInput({ error }: { error?: string }) {
  const [value, setValue] = useState(0);
  const [hovered, setHovered] = useState(0);
  const errorId = useId();
  const displayed = hovered || value;

  return (
    <fieldset aria-describedby={error ? errorId : undefined}>
      <legend className="mb-2 text-sm font-medium">Note sur 5</legend>
      <div className="flex gap-1" onMouseLeave={() => setHovered(0)}>
        {[1, 2, 3, 4, 5].map((star) => (
          <label
            key={star}
            onMouseEnter={() => setHovered(star)}
            className="inline-flex size-11 cursor-pointer items-center justify-center rounded-lg has-focus-visible:outline-2 has-focus-visible:outline-offset-2 has-focus-visible:outline-primary"
          >
            <input
              type="radio"
              name="rating"
              value={star}
              checked={value === star}
              onChange={() => setValue(star)}
              className="sr-only"
            />
            <svg
              viewBox="0 0 20 20"
              aria-hidden="true"
              className={cn(
                "size-8 transition-colors motion-reduce:transition-none",
                star <= displayed ? "fill-rating" : "fill-border",
              )}
            >
              <path d={STAR_PATH} />
            </svg>
            <span className="sr-only">
              {star} étoile{star > 1 ? "s" : ""} sur 5
            </span>
          </label>
        ))}
      </div>
      {error && (
        <p id={errorId} className="mt-2 text-sm text-danger">
          {error}
        </p>
      )}
    </fieldset>
  );
}

export function RatingForm({ incidentId, heroAlias }: { incidentId: string; heroAlias: string }) {
  const [state, formAction] = useActionState(
    rateInterventionAction,
    idleFormState as FormState<"rating" | "comment">,
  );

  return (
    <form action={formAction} noValidate className="space-y-4">
      <input type="hidden" name="incidentId" value={incidentId} />
      <StarRatingInput error={state.fieldErrors.rating} />
      <TextAreaField
        label="Commentaire (facultatif)"
        name="comment"
        rows={3}
        maxLength={500}
        placeholder={`Comment s'est passée l'intervention de ${heroAlias} ?`}
        defaultValue={state.values?.comment}
        error={state.fieldErrors.comment}
      />
      <FormMessage state={state} />
      <SubmitButton pendingLabel="Envoi…">Envoyer mon avis</SubmitButton>
    </form>
  );
}
