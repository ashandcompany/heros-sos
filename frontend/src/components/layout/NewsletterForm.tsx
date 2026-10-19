"use client";

import { useActionState, useId } from "react";
import { subscribeToNewsletterAction, type NewsletterFormState } from "@/app/actions/newsletter";
import { buttonClasses } from "@/components/ui/ButtonLink";
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
      <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
        <span className="font-semibold text-success">Inscription confirmée !</span> Vous recevrez
        désormais nos alertes et actualités héroïques.
      </p>
    );
  }

  if (alreadySubscribed) {
    return <p className="text-sm text-muted">Vous êtes inscrit·e à notre newsletter. Merci !</p>;
  }

  const hasError = state.status === "error";

  return (
    <form action={formAction} noValidate>
      <label htmlFor={inputId} className="mb-2 block text-sm font-medium">
        Adresse email
      </label>
      <div className="flex flex-col gap-3 sm:flex-row">
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
            "min-h-11 w-full rounded-full border bg-background px-4 text-foreground placeholder:text-muted",
            hasError ? "border-danger" : "border-border",
          )}
        />
        <button
          type="submit"
          disabled={isPending}
          className={cn(buttonClasses("primary"), "shrink-0 disabled:opacity-60")}
        >
          {isPending ? "Inscription…" : "S'inscrire"}
        </button>
      </div>
      <p id={errorId} role="alert" className="mt-2 min-h-5 text-sm text-danger">
        {hasError ? state.message : null}
      </p>
    </form>
  );
}
