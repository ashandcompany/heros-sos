"use client";

import { useActionState } from "react";
import { resetPasswordAction, type ResetPasswordFormState } from "@/app/actions/password";
import { ButtonLink, buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { PasswordField } from "@/components/ui/PasswordField";
import { routes } from "@/lib/routes";
import { PASSWORD_RULES } from "@/lib/validation/signup";

const initialState: ResetPasswordFormState = { status: "idle" };

/** Nouveau mot de passe, avec le jeton du lien reçu par mail (utilisable une seule fois). */
export function ResetPasswordForm({ token }: { token: string }) {
  const [state, formAction, isPending] = useActionState(resetPasswordAction, initialState);

  if (state.status === "success") {
    return (
      <div className="space-y-6">
        <p role="status" className="rounded-xl border border-success/40 bg-success/10 p-4 text-sm">
          <span className="block font-semibold text-success">Mot de passe modifié.</span>
          Par sécurité, vous avez été déconnecté de tous vos appareils. Connectez-vous avec votre
          nouveau mot de passe.
        </p>
        <ButtonLink href={routes.login} size="lg" className="w-full">
          Se connecter
        </ButtonLink>
      </div>
    );
  }

  if (state.status === "invalid-token") return <InvalidResetLink />;

  const errors = state.status === "error" ? state.fieldErrors : {};

  return (
    <form action={formAction} noValidate className="space-y-5">
      <input type="hidden" name="token" value={token} />
      {state.status === "error" && state.message && (
        <p
          role="alert"
          className="rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger"
        >
          {state.message}
        </p>
      )}
      <PasswordField
        label="Nouveau mot de passe"
        hint={PASSWORD_RULES}
        name="password"
        autoComplete="new-password"
        required
        error={errors.password}
      />
      <PasswordField
        label="Confirmer le mot de passe"
        name="passwordConfirmation"
        autoComplete="new-password"
        required
        error={errors.passwordConfirmation}
      />
      <button
        type="submit"
        disabled={isPending}
        className={cn(buttonClasses("primary", "lg"), "w-full disabled:opacity-60")}
      >
        {isPending ? "Enregistrement…" : "Changer mon mot de passe"}
      </button>
    </form>
  );
}

/** Lien absent, incomplet, expiré ou déjà utilisé : il faut en demander un nouveau. */
export function InvalidResetLink() {
  return (
    <div className="space-y-6">
      <p role="alert" className="rounded-xl border border-danger/50 bg-danger/10 p-4 text-sm">
        <span className="block font-semibold text-danger">Lien invalide ou expiré.</span>
        Ce lien est incomplet, a déjà servi ou date de plus d&apos;une heure. Demandez-en un
        nouveau.
      </p>
      <ButtonLink href={routes.forgotPassword} variant="secondary" size="lg" className="w-full">
        Recevoir un nouveau lien
      </ButtonLink>
    </div>
  );
}
