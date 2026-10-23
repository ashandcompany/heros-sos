import Link from "next/link";
import {
  updateModeratorAvatarAction,
  updateModeratorContactAction,
  updateModeratorNewsletterAction,
} from "@/app/actions/admin";
import { Panel } from "@/components/citizen/AccountDetails";
import { AvatarForm, ContactForm, NewsletterSettingsForm } from "@/components/citizen/AccountForms";
import { Avatar } from "@/components/ui/Avatar";
import { getMyAccount } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import { civilityOptions } from "@/lib/validation/signup";

/** Contenu de « Mon profil » du back-office. Lit la session : à rendre sous <Suspense>. */
export async function ModeratorProfileDetails() {
  const token = await requireAdminSession();
  const account = await getMyAccount(token);
  const civility = civilityOptions.find(({ value }) => value === account.civility)?.label;
  const fullName = `${account.firstName} ${account.lastName}`;
  const notProvided = <span className="text-muted">Non communiqué</span>;
  // Un compte modérateur est créé hors ligne sans adresse : le formulaire part alors de zéro.
  const address = account.address.street ? account.address : null;

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
          <AvatarForm action={updateModeratorAvatarAction} />
        </Panel>

        <Panel
          id="identity-title"
          title="Informations personnelles"
          description="Elles se modifient hors ligne, par l'équipe technique."
        >
          <dl className="space-y-3">
            {[
              ["Civilité", civility],
              ["Nom", account.lastName],
              ["Prénom", account.firstName],
              ["Date de naissance", account.birthDate && formatDate(account.birthDate)],
            ].map(([label, value]) => (
              <div key={label}>
                <dt className="text-sm text-muted">{label}</dt>
                <dd>{value || notProvided}</dd>
              </div>
            ))}
          </dl>
        </Panel>
      </div>

      <div className="space-y-6">
        <Panel id="contact-title" title="Coordonnées">
          <ContactForm
            contact={{ mobilePhone: account.mobilePhone, address }}
            action={updateModeratorContactAction}
          />
        </Panel>

        <Panel
          id="email-title"
          title="Adresse email"
          description="Celle d'un compte de modérateur se change hors ligne, par l'équipe technique : aucun autre modérateur n'a à valider la demande."
        >
          <p className="text-sm text-muted">Adresse actuelle</p>
          <p className="font-medium break-all">{account.email}</p>
        </Panel>

        <Panel id="newsletter-settings-title" title="Newsletter">
          <NewsletterSettingsForm
            subscribed={account.newsletterSubscribed}
            action={updateModeratorNewsletterAction}
          />
        </Panel>

        <p className="text-sm text-muted">
          Double authentification et sessions ouvertes :{" "}
          <Link href={routes.adminSecurity} className="font-medium text-foreground underline">
            page Sécurité
          </Link>
          .
        </p>
      </div>
    </div>
  );
}
