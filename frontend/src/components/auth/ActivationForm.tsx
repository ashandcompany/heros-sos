"use client";

import { useActionState } from "react";
import {
  activateAccountAction,
  resendActivationAction,
  type ActivationFormState,
  type ResendActivationFormState,
} from "@/app/actions/activation";
import { ButtonLink, buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { TextField } from "@/components/ui/TextField";
import { routes } from "@/lib/routes";

const initialActivationState: ActivationFormState = { status: "idle" };
const initialResendState: ResendActivationFormState = { status: "idle" };

/**
 * Activation par un bouton plutôt qu'au chargement de la page : les messageries et antivirus qui
 * ouvrent les liens des mails à l'avance consommeraient sinon le jeton (utilisable une seule fois).
 */
export function ActivationForm({ token }: { token: string }) {
  const [state, formAction, isPending] = useActionState(
    activateAccountAction,
    initialActivationState,
  );

  if (state.status === "success") {
    return (
      <div className="space-y-6">
        <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
          <span className="block font-semibold text-success">Compte activé.</span>
          Vous pouvez maintenant vous connecter avec votre adresse email et votre mot de passe.
        </p>
        <ButtonLink href={routes.login} size="lg" className="w-full">
          Se connecter
        </ButtonLink>
      </div>
    );
  }

  if (state.status === "error" && state.error === "invalid-token") {
    return (
      <div className="space-y-6">
        <p role="alert" className="rounded-xl border border-danger/50 bg-danger/10 p-4 text-sm">
          <span className="block font-semibold text-danger">Lien invalide ou expiré.</span>
          Ce lien a peut-être déjà servi (votre compte est alors activé : essayez de vous
          connecter), ou il date de plus de 24 heures. Demandez un nouveau lien ci-dessous.
        </p>
        <ResendActivationForm />
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
          L&apos;activation est momentanément indisponible. Réessayez dans quelques instants.
        </p>
      )}
      <button
        type="submit"
        disabled={isPending}
        className={cn(buttonClasses("primary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? "Activation…" : "Activer mon compte"}
      </button>
    </form>
  );
}

/** Demande d'un nouveau lien d'activation, envoyé par mail. */
export function ResendActivationForm() {
  const [state, formAction, isPending] = useActionState(resendActivationAction, initialResendState);

  if (state.status === "success") {
    return (
      <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
        <span className="block font-semibold text-success">Demande envoyée.</span>
        Si un compte en attente d&apos;activation est associé à cette adresse, vous recevrez un
        nouveau lien. Pensez à vérifier vos courriers indésirables.
      </p>
    );
  }

  const hasError = state.status === "error";

  return (
    <form action={formAction} noValidate className="space-y-5">
      <TextField
        label="Adresse email du compte"
        name="email"
        type="email"
        inputMode="email"
        autoComplete="email"
        autoCapitalize="none"
        spellCheck={false}
        required
        defaultValue={hasError ? state.email : undefined}
        error={hasError ? state.message : undefined}
      />
      <button
        type="submit"
        disabled={isPending}
        className={cn(buttonClasses("secondary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? "Envoi…" : "Recevoir un nouveau lien"}
      </button>
    </form>
  );
}
