"use client";

import { useActionState } from "react";
import { ButtonLink, buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import type { TokenFormState } from "@/lib/validation/form-state";

interface Outcome {
  title: string;
  message: string;
}

interface TokenConfirmationFormProps {
  /** Jeton du lien reçu par mail (`?token=…`), renvoyé tel quel à l'action. */
  token: string;
  action: (previousState: TokenFormState, formData: FormData) => Promise<TokenFormState>;
  submitLabel: string;
  pendingLabel: string;
  success: Outcome;
  invalid: Outcome;
  /** Message d'une panne (API injoignable, erreur inattendue) : le lien reste utilisable. */
  unavailable: string;
  /** Où poursuivre une fois l'action faite ou le lien refusé. */
  continueHref: string;
  continueLabel: string;
}

/**
 * Action déclenchée par un LIEN reçu par mail, confirmée par un bouton plutôt qu'au chargement de la page :
 * les messageries et antivirus qui ouvrent les liens des mails à l'avance agiraient sinon à la place de la
 * personne (abonner, désabonner, confirmer une adresse). Sert à toutes les pages de ce genre.
 */
export function TokenConfirmationForm({
  token,
  action,
  submitLabel,
  pendingLabel,
  success,
  invalid,
  unavailable,
  continueHref,
  continueLabel,
}: TokenConfirmationFormProps) {
  const [state, formAction, isPending] = useActionState(action, {
    status: "idle",
  } as TokenFormState);

  if (state.status === "success") {
    return (
      <div className="space-y-6">
        <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
          <span className="block font-semibold text-success">{success.title}</span>
          {success.message}
        </p>
        <ButtonLink href={continueHref} size="lg" className="w-full">
          {continueLabel}
        </ButtonLink>
      </div>
    );
  }

  if (state.status === "error" && state.error === "invalid-token") {
    return (
      <div className="space-y-6">
        <p role="alert" className="rounded-xl border border-danger/50 bg-danger/10 p-4 text-sm">
          <span className="block font-semibold text-danger">{invalid.title}</span>
          {invalid.message}
        </p>
        <ButtonLink href={continueHref} variant="secondary" size="lg" className="w-full">
          {continueLabel}
        </ButtonLink>
      </div>
    );
  }

  return (
    <form action={formAction} className="space-y-5">
      <input type="hidden" name="token" value={token} />
      {state.status === "error" && (
        <p
          role="alert"
          className="rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger"
        >
          {unavailable}
        </p>
      )}
      <button
        type="submit"
        disabled={isPending}
        className={cn(buttonClasses("primary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? pendingLabel : submitLabel}
      </button>
    </form>
  );
}
