import type { ReactNode } from "react";
import { Avatar } from "@/components/ui/Avatar";
import { getMyAccount } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { requireCitizenSession } from "@/lib/server-session";
import { civilityOptions } from "@/lib/validation/signup";
import type { MyAccount } from "@/types";
import { AvatarForm, ContactForm, EmailChangeForm, NewsletterSettingsForm } from "./AccountForms";

export function Panel({
  id,
  title,
  description,
  children,
}: {
  id: string;
  title: string;
  description?: string;
  children: ReactNode;
}) {
  return (
    <section
      aria-labelledby={id}
      className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8"
    >
      <h2 id={id} className="font-display text-xl font-semibold">
        {title}
      </h2>
      {description && <p className="mt-1 text-sm text-muted">{description}</p>}
      <div className="mt-6">{children}</div>
    </section>
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
