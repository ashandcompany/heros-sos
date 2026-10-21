"use client";

import { useActionState } from "react";
import {
  enableMfaAction,
  startMfaSetupAction,
  type MfaRecoveryState,
  type MfaSetupState,
} from "@/app/actions/mfa";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextField } from "@/components/ui/TextField";
import { idleFormState } from "@/lib/validation/form-state";
import { RecoveryCodesNotice } from "./RecoveryCodesNotice";

const initialSetup: MfaSetupState = { status: "idle" };

/** « JBSWY3DP EHPK3PXP » : le secret par groupes de quatre, plus facile à recopier. */
const groupSecret = (secret: string) => secret.match(/.{1,4}/g)?.join(" ") ?? secret;

/**
 * Activation de la double authentification : un secret à saisir dans une application d'authentification,
 * puis un premier code qui la confirme et rend les codes de secours. Sert à la connexion (activation
 * obligatoire) et à la page de sécurité (activation volontaire).
 */
export function MfaSetup({
  continueHref,
  continueLabel,
}: {
  /** Où poursuivre une fois les codes de secours notés. */
  continueHref: string;
  continueLabel?: string;
}) {
  const [setup, startSetup] = useActionState(startMfaSetupAction, initialSetup);
  const [enabled, enable] = useActionState(enableMfaAction, idleFormState as MfaRecoveryState);

  if (enabled.status === "success" && enabled.recoveryCodes) {
    return (
      <RecoveryCodesNotice
        title="Double authentification activée."
        codes={enabled.recoveryCodes}
        continueHref={continueHref}
        continueLabel={continueLabel}
      />
    );
  }

  if (setup.status !== "ready") {
    return (
      <form action={startSetup} className="space-y-4">
        {setup.status === "error" && (
          <p
            role="alert"
            className="rounded-xl border border-danger/50 bg-danger/10 px-4 py-3 text-sm text-danger"
          >
            {setup.message}
          </p>
        )}
        <SubmitButton pendingLabel="Préparation…">Commencer l&apos;activation</SubmitButton>
      </form>
    );
  }

  return (
    <div className="space-y-6">
      <ol className="list-decimal space-y-3 pl-5 text-sm">
        <li>
          Installez une application d&apos;authentification sur votre téléphone (Google
          Authenticator, Aegis, 1Password, Authy…).
        </li>
        <li>
          Ajoutez-y un compte avec cette clé, ou ouvrez ce lien depuis votre téléphone :
          <p className="mt-2 border-2 border-border bg-background px-4 py-3 font-mono text-base tracking-wider break-all">
            <span aria-label="Clé de configuration" className="select-all">
              {groupSecret(setup.secret)}
            </span>
          </p>
          <a href={setup.otpauthUrl} className="mt-2 inline-block text-accent hover:underline">
            Ouvrir dans mon application d&apos;authentification
          </a>
        </li>
        <li>Saisissez ci-dessous le code à 6 chiffres qu&apos;elle affiche.</li>
      </ol>

      <form action={enable} noValidate className="space-y-4">
        <FormMessage state={enabled} />
        <TextField
          label="Code à 6 chiffres"
          name="code"
          inputMode="numeric"
          autoComplete="one-time-code"
          spellCheck={false}
          maxLength={7}
          required
          error={enabled.fieldErrors.code}
        />
        <SubmitButton pendingLabel="Vérification…">Activer la double authentification</SubmitButton>
      </form>
    </div>
  );
}
