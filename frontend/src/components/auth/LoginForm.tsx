"use client";

import Link from "next/link";
import { useActionState } from "react";
import { loginAction, type LoginFormState } from "@/app/actions/auth";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { PasswordField } from "@/components/ui/PasswordField";
import { TextField } from "@/components/ui/TextField";
import { routes } from "@/lib/routes";

const initialState: LoginFormState = { identifier: "", fieldErrors: {} };

export function LoginForm() {
  const [state, formAction, isPending] = useActionState(loginAction, initialState);

  return (
    <form action={formAction} noValidate className="space-y-5">
      {state.formError && (
        <p
          role="alert"
          className="rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger"
        >
          {state.formError}
        </p>
      )}

      <TextField
        label="Adresse email ou pseudo"
        hint="La connexion par pseudo est réservée aux super-héros."
        name="identifier"
        type="text"
        autoComplete="username"
        autoCapitalize="none"
        spellCheck={false}
        required
        defaultValue={state.identifier}
        error={state.fieldErrors.identifier}
      />

      <PasswordField
        label="Mot de passe"
        name="password"
        autoComplete="current-password"
        required
        error={state.fieldErrors.password}
        labelAside={
          <Link
            href={routes.forgotPassword}
            className="rounded-sm text-sm font-medium text-accent hover:underline"
          >
            Mot de passe oublié&nbsp;?
          </Link>
        }
      />

      <button
        type="submit"
        disabled={isPending}
        className={cn(buttonClasses("primary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? "Connexion…" : "Se connecter"}
      </button>

      <p role="status" className="sr-only">
        {isPending ? "Connexion en cours" : ""}
      </p>
    </form>
  );
}
