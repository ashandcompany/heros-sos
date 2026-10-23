import { MfaSection } from "@/components/auth/MfaPanel";
import { Avatar } from "@/components/ui/Avatar";
import { getMyAccount } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { requireCitizenSession } from "@/lib/server-session";
import { civilityOptions } from "@/lib/validation/signup";
import type { MyAccount } from "@/types";
import {
  AvatarForm,
  ContactForm,
  EmailChangeForm,
  LogoutAllForm,
  NewsletterSettingsForm,
  PasswordChangeForm,
  RemoveAvatarForm,
} from "./AccountForms";
import { Panel } from "./Panel";

export { Panel };

/** Panneau « Sessions » : même texte pour tous les rôles (citoyen, héros, modérateur). */
export function SessionsPanel() {
  return (
    <Panel
      id="sessions-title"
      title="Sessions"
      description="Ferme toutes vos sessions ouvertes, sur tous vos appareils, celle-ci comprise. Utile si vous avez oublié de vous déconnecter d'un ordinateur partagé ou si vous pensez que quelqu'un d'autre a accès à votre compte."
    >
      <LogoutAllForm />
    </Panel>
  );
}

/** Panneau « Mot de passe » : même formulaire pour tous les rôles. */
export function PasswordPanel() {
  return (
    <Panel
      id="password-title"
      title="Mot de passe"
      description="Après le changement, vous restez connecté ici ; vos autres appareils sont déconnectés."
    >
      <PasswordChangeForm />
    </Panel>
  );
}

/** Contenu de « Mon compte ». Lit la session : à rendre sous <Suspense>. */
export async function AccountDetails() {
  const token = await requireCitizenSession();
  const account = await getMyAccount(token);
  return <AccountPanels account={account} />;
}

function AccountPanels({ account }: { account: MyAccount }) {
  const civility = civilityOptions.find(({ value }) => value === account.civility)?.label;
  const fullName = `${account.firstName} ${account.lastName}`;

  return (
    <div className="grid gap-6 lg:grid-cols-[20rem_1fr] lg:items-start">
      <div className="space-y-6">
        <Panel id="photo-title" title="Photo de profil">
          <div className="mb-6 flex items-center gap-4">
            <Avatar name={account.firstName} src={account.avatarUrl} size={80} />
            <div>
              <p className="font-display text-lg font-semibold">{fullName}</p>
              <p className="text-sm text-muted">
                Membre depuis le{" "}
                <time dateTime={account.registeredAt}>{formatDate(account.registeredAt)}</time>
              </p>
            </div>
          </div>
          <AvatarForm />
          {account.avatarUrl && <RemoveAvatarForm />}
        </Panel>

        <Panel
          id="identity-title"
          title="Informations personnelles"
          description="Ces informations ne sont pas modifiables depuis votre espace."
        >
          <dl className="space-y-3">
            {[
              ["Civilité", civility],
              ["Nom", account.lastName],
              ["Prénom", account.firstName],
              ["Date de naissance", formatDate(account.birthDate)],
            ].map(([label, value]) => (
              <div key={label}>
                <dt className="text-sm text-muted">{label}</dt>
                <dd>{value}</dd>
              </div>
            ))}
          </dl>
        </Panel>
      </div>

      <div className="space-y-6">
        <Panel id="contact-title" title="Coordonnées">
          <ContactForm contact={account} />
        </Panel>

        <Panel id="email-title" title="Adresse email">
          <p className="text-sm text-muted">Adresse actuelle</p>
          <p className="mb-4 font-medium break-all">{account.email}</p>
          {account.pendingEmail && (
            <p className="mb-6 rounded-xl border border-rating/40 bg-rating/10 px-4 py-3 text-sm">
              <span className="font-semibold text-rating">
                {account.pendingEmailConfirmed
                  ? "En attente de modération :"
                  : "En attente de confirmation :"}
              </span>{" "}
              <span className="break-all">{account.pendingEmail}</span>
              {!account.pendingEmailConfirmed && (
                <span className="mt-1 block text-muted">
                  Ouvrez le lien du mail envoyé à cette adresse. Pas reçu ? Renvoyez la demande
                  ci-dessous.
                </span>
              )}
            </p>
          )}
          <EmailChangeForm />
        </Panel>

        <Panel id="newsletter-settings-title" title="Newsletter">
          <NewsletterSettingsForm subscribed={account.newsletterSubscribed} />
        </Panel>

        <PasswordPanel />

        <MfaSection role="citoyen" />

        <SessionsPanel />
      </div>
    </div>
  );
}

export function AccountDetailsSkeleton() {
  return (
    <div aria-hidden="true" className="grid gap-6 lg:grid-cols-[20rem_1fr]">
      <div className="h-96 border-2 border-border bg-surface motion-safe:animate-pulse" />
      <div className="h-[36rem] border-2 border-border bg-surface motion-safe:animate-pulse" />
    </div>
  );
}
