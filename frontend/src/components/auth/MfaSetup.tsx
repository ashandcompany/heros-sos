"use client";

import { QRCodeSVG } from "qrcode.react";
import { useActionState } from "react";
import {
  enableMfaAction,
  startMfaSetupAction,
  type MfaRecoveryState,
  type MfaSetupState,
} from "@/app/actions/mfa";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { FormMessage } from "@/components/ui/FormMessage";
import { SubmitButton } from "@/components/ui/SubmitButton";
import { TextField } from "@/components/ui/TextField";
import { idleFormState } from "@/lib/validation/form-state";
import { CopyButton } from "./CopyButton";
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
      <ol className="list-decimal space-y-4 pl-5 text-sm">
        <li>
          Installez une application d&apos;authentification sur l&apos;appareil qui vous servira à
          vous connecter (Mots de passe d&apos;Apple, Google Authenticator, Aegis, 1Password,
          Authy…).
        </li>
        <li>
          Ajoutez-y votre compte, au choix :
          <div className="mt-3 space-y-5">
            <div>
              <p className="font-semibold">Depuis cet appareil</p>
              {/* Deux liens, car Apple n'ouvre pas le lien standard : son app Mots de passe n'attend que
                  `apple-otpauth:` (voir lib/otpauth.ts). Le lien standard sert aux autres applications. */}
              <div className="mt-2 flex flex-wrap gap-3">
                <a href={setup.appleOtpauthUrl} className={buttonClasses("primary")}>
                  Ajouter à Mots de passe (Apple)
                </a>
                <a href={setup.otpauthUrl} className={buttonClasses("secondary")}>
                  Autre application
                </a>
              </div>
              <p className="mt-2 text-muted">
                Sur iPhone, iPad ou Mac, le premier bouton ouvre Mots de passe, qui propose le
                compte enregistré pour ce site (celui de votre connexion, si votre navigateur
                l&apos;a enregistré dans le Trousseau iCloud) et y ajoute le code. Mots de passe ne
                range un code que dans un compte : sans compte enregistré pour ce site, créez-le
                d&apos;abord dans Mots de passe, ou utilisez une application
                d&apos;authentification, qui crée l&apos;entrée toute seule. Le second bouton ouvre
                celle que l&apos;appareil a choisie (Google Authenticator, Aegis, 1Password,
                Authy…). Rien ne s&apos;ouvre ? Prenez l&apos;une des deux autres méthodes.
              </p>
            </div>

            <div>
              <p className="font-semibold">Avec un autre appareil</p>
              <p>Dans l&apos;application de votre téléphone, scannez ce QR code :</p>
              {/* Dessiné ici, dans le navigateur : l'adresse otpauth contient le secret, elle ne doit jamais
                  passer par un service de QR code. Fond blanc et marge fixes, quel que soit le thème : un QR
                  code se lit sombre sur clair. */}
              <div className="mt-2 block w-fit border-2 border-border bg-white">
                <QRCodeSVG
                  value={setup.otpauthUrl}
                  size={192}
                  level="M"
                  marginSize={3}
                  bgColor="#ffffff"
                  fgColor="#000000"
                  role="img"
                  aria-label="QR code de configuration de la double authentification"
                  className="block"
                />
              </div>
              <p className="mt-2 text-muted">
                Avec Mots de passe d&apos;Apple : ouvrez le compte de ce site, Modifier, Configurer
                un code (« Set Up Code »), puis scannez-le depuis l&apos;application.
              </p>
            </div>

            <div>
              <p className="font-semibold">À la main</p>
              <p>Saisissez cette clé dans l&apos;application (code basé sur le temps) :</p>
              <p className="mt-2 border-2 border-border bg-background px-4 py-3 font-mono text-base tracking-wider break-all">
                <span aria-label="Clé de configuration" className="select-all">
                  {groupSecret(setup.secret)}
                </span>
              </p>
              <div className="mt-2">
                <CopyButton text={setup.secret} label="Copier la clé" copiedLabel="Clé copiée" />
              </div>
              <p className="mt-2 text-muted">
                Avec Mots de passe d&apos;Apple : ouvrez le compte de ce site, Modifier, Configurer
                un code (« Set Up Code »), Saisir la clé (« Enter Setup Key »), collez-la, puis
                Utiliser la clé (« Use Setup Key »).
              </p>
            </div>
          </div>
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
