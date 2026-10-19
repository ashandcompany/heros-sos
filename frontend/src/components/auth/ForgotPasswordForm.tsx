"use client";

import { useActionState } from "react";
import { forgotPasswordAction, type ForgotPasswordFormState } from "@/app/actions/password";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { TextField } from "@/components/ui/TextField";

const initialState: ForgotPasswordFormState = { status: "idle" };

export function ForgotPasswordForm() {
  const [state, formAction, isPending] = useActionState(forgotPasswordAction, initialState);

  if (state.status === "success") {
    return (
      <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
        <span className="block font-semibold text-success">Demande envoyée.</span>
        Si un compte est associé à cette adresse, vous recevrez un email contenant un lien pour
        réinitialiser votre mot de passe. Pensez à vérifier vos courriers indésirables.
      </p>
    );
  }

  const hasError = state.status === "error";

  return (
    <form action={formAction} noValidate className="space-y-5">
      <TextField
        label="Adresse email"
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
        className={cn(buttonClasses("primary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? "Envoi…" : "Envoyer le lien de réinitialisation"}
      </button>
    </form>
  );
}
