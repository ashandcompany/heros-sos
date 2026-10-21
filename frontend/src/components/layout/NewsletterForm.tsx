"use client";

import { useActionState, useId } from "react";
import { subscribeToNewsletterAction, type NewsletterFormState } from "@/app/actions/newsletter";
import { cn } from "@/components/ui/cn";

const initialState: NewsletterFormState = { status: "idle" };

interface NewsletterFormProps {
  /** `true` si le cookie d'inscription est déjà présent : le formulaire n'est pas affiché. */
  alreadySubscribed: boolean;
}

export function NewsletterForm({ alreadySubscribed }: NewsletterFormProps) {
  const [state, formAction, isPending] = useActionState(subscribeToNewsletterAction, initialState);
  const inputId = useId();
  const errorId = useId();

  if (state.status === "success") {
    return (
      <p role="status" className="border-2 border-dashed border-background p-4 text-sm">
        <span className="font-semibold">Demande enregistrée.</span> Confirmez votre abonnement avec
        le lien envoyé à cette adresse (valable 48 heures) : aucun bulletin ne part avant.
      </p>
    );
  }

  if (alreadySubscribed) {
    return (
      <p className="text-sm text-ink/85">
        Une demande d&apos;abonnement est déjà enregistrée à votre nom. Merci.
      </p>
    );
  }

  const hasError = state.status === "error";

  return (
    <form action={formAction} noValidate>
      <label
        htmlFor={inputId}
        className="mb-1 block font-mono text-xs font-medium tracking-widest uppercase"
      >
        Adresse email du demandeur
      </label>
      <div className="flex flex-col gap-4 sm:flex-row sm:items-end">
        <input
          id={inputId}
          name="email"
          type="email"
          autoComplete="email"
          inputMode="email"
          required
          placeholder="vous@exemple.fr"
          defaultValue={hasError ? state.email : undefined}
          aria-invalid={hasError}
          aria-describedby={hasError ? errorId : undefined}
          className={cn(
            "min-h-11 w-full rounded-none border-0 border-b-2 bg-transparent px-1 font-mono text-ink placeholder:text-ink/60",
            hasError ? "border-brick" : "border-background",
          )}
        />
        <button
          type="submit"
          disabled={isPending}
          className="min-h-11 shrink-0 rounded-sm border-[3px] border-double border-brick px-4 font-stamp tracking-[0.18em] text-brick uppercase hover:bg-brick hover:text-paper active:translate-x-px active:translate-y-px disabled:opacity-60"
        >
          {isPending ? "Tamponnage…" : "Tamponner"}
        </button>
      </div>
      <p id={errorId} role="alert" className="mt-2 min-h-5 text-sm font-semibold text-brick">
        {hasError ? state.message : null}
      </p>
    </form>
  );
}
