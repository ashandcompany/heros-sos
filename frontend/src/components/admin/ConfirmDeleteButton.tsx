"use client";

import { useActionState, useId, useRef } from "react";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { idleFormState, type FormState } from "@/lib/validation/form-state";

type DeleteAction = (previousState: FormState, formData: FormData) => Promise<FormState>;

/** Bouton « Supprimer » qui demande confirmation dans un <dialog> natif avant d'agir. */
export function ConfirmDeleteButton({
  action,
  id,
  subject,
  description,
}: {
  action: DeleteAction;
  id: string;
  /** Nom de l'élément supprimé (ex : titre de l'article). */
  subject: string;
  description: string;
}) {
  const [state, formAction] = useActionState(action, idleFormState);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  return (
    <>
      <button
        type="button"
        aria-haspopup="dialog"
        onClick={() => dialogRef.current?.showModal()}
        className={buttonClasses("danger")}
      >
        Supprimer<span className="sr-only"> : {subject}</span>
      </button>

      <dialog
        ref={dialogRef}
        aria-labelledby={titleId}
        className="m-auto w-[calc(100%-2rem)] max-w-md rounded-2xl border border-border bg-surface p-6 text-foreground backdrop:bg-black/70 backdrop:backdrop-blur-sm sm:p-8"
      >
        <h2 id={titleId} className="font-display text-xl font-bold">
          Supprimer «&nbsp;{subject}&nbsp;»&nbsp;?
        </h2>
        <p className="mt-2 text-sm text-muted">{description}</p>
        <div className="mt-6 space-y-4">
          <FormMessage state={state} />
          <div className="flex flex-wrap justify-end gap-2">
            <button
              type="button"
              onClick={() => dialogRef.current?.close()}
              className={cn(buttonClasses("secondary"))}
            >
              Annuler
            </button>
            <form action={formAction}>
              <input type="hidden" name="id" value={id} />
              <SubmitButton variant="danger" pendingLabel="Suppression…">
                Supprimer définitivement
              </SubmitButton>
            </form>
          </div>
        </div>
      </dialog>
    </>
  );
}
