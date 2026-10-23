"use client";

import { useActionState, useId, useRef } from "react";
import { sendNewsletterAction } from "@/app/actions/admin";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { idleFormState } from "@/lib/validation/form-state";

/** Bouton « Envoyer » qui demande confirmation dans un <dialog> natif : l'envoi est définitif. */
export function SendNewsletterButton({
  id,
  subject,
  targeted,
}: {
  id: string;
  subject: string;
  /** Vrai si le brouillon vise un seul compte plutôt que tous les abonnés. */
  targeted: boolean;
}) {
  const [state, formAction] = useActionState(sendNewsletterAction, idleFormState);
  const dialogRef = useRef<HTMLDialogElement>(null);
  const titleId = useId();

  return (
    <>
      <button
        type="button"
        aria-haspopup="dialog"
        onClick={() => dialogRef.current?.showModal()}
        className={buttonClasses("primary")}
      >
        Envoyer<span className="sr-only"> : {subject}</span>
      </button>

      <dialog
        ref={dialogRef}
        aria-labelledby={titleId}
        className="m-auto w-[calc(100%-2rem)] max-w-md border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 text-foreground backdrop:bg-black/70 backdrop:backdrop-blur-sm sm:p-8"
      >
        <h2 id={titleId} className="font-display text-xl font-bold">
          Envoyer «&nbsp;{subject}&nbsp;»&nbsp;?
        </h2>
        <p className="mt-2 text-sm text-muted">
          {targeted
            ? "Le mail part vers le compte destinataire de ce brouillon, s'il est abonné."
            : "Le mail part vers tous les abonnés confirmés de la newsletter."}{" "}
          Une newsletter envoyée ne peut plus être modifiée ni renvoyée.
        </p>
        <div className="mt-6 space-y-4">
          <FormMessage state={state} />
          <div className="flex flex-wrap justify-end gap-2">
            <button
              type="button"
              onClick={() => dialogRef.current?.close()}
              className={buttonClasses("secondary")}
            >
              Annuler
            </button>
            <form action={formAction}>
              <input type="hidden" name="id" value={id} />
              <SubmitButton pendingLabel="Envoi…">Envoyer maintenant</SubmitButton>
            </form>
          </div>
        </div>
      </dialog>
    </>
  );
}
