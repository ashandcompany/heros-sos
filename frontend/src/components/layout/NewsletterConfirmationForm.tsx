"use client";

import { useActionState } from "react";
import {
  confirmNewsletterAction,
  type NewsletterConfirmationFormState,
} from "@/app/actions/newsletter";
import { ButtonLink, buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { routes } from "@/lib/routes";

const initialState: NewsletterConfirmationFormState = { status: "idle" };

/**
 * Confirmation par un bouton plutôt qu'au chargement de la page : les messageries et antivirus qui
 * ouvrent les liens des mails à l'avance donneraient sinon le consentement à la place du visiteur.
 */
export function NewsletterConfirmationForm({ token }: { token: string }) {
  const [state, formAction, isPending] = useActionState(confirmNewsletterAction, initialState);

  if (state.status === "success") {
    return (
      <div className="space-y-6">
        <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
          <span className="block font-semibold text-success">Abonnement confirmé.</span>
          Vous recevrez désormais le bulletin du Bureau.
        </p>
        <ButtonLink href={routes.home} size="lg" className="w-full">
          Retour à l&apos;accueil
        </ButtonLink>
      </div>
    );
  }

  if (state.status === "error" && state.error === "invalid-token") {
    return (
      <p role="alert" className="rounded-xl border border-danger/50 bg-danger/10 p-4 text-sm">
        <span className="block font-semibold text-danger">Lien invalide ou expiré.</span>
        Ce lien date peut-être de plus de 48 heures. Inscrivez-vous de nouveau depuis le bas de
        n&apos;importe quelle page pour en recevoir un autre.
      </p>
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
          La confirmation est momentanément indisponible. Réessayez dans quelques instants.
        </p>
      )}
      <button
        type="submit"
        disabled={isPending}
        className={cn(buttonClasses("primary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? "Confirmation…" : "Confirmer mon abonnement"}
      </button>
    </form>
  );
}
