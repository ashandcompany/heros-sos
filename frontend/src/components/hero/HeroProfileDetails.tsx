import {
  requestHeroEmailChangeAction,
  updateHeroAvatarAction,
  updateHeroContactAction,
} from "@/app/actions/hero";
import { Panel } from "@/components/citizen/AccountDetails";
import { AvatarForm, ContactForm, EmailChangeForm } from "@/components/citizen/AccountForms";
import { Avatar } from "@/components/ui/Avatar";
import { getMyHeroProfile } from "@/lib/api";
import { formatDate, formatFullName } from "@/lib/format";
import { requireHeroSession } from "@/lib/server-session";
import { civilityOptions } from "@/lib/validation/signup";

/** Contenu de « Mon profil ». Lit la session : à rendre sous <Suspense>. */
export async function HeroProfileDetails() {
  const { token } = await requireHeroSession();
  const profile = await getMyHeroProfile(token);
  const civility = civilityOptions.find(({ value }) => value === profile.civility)?.label;
  const notProvided = <span className="text-muted">Non communiqué</span>;

  return (
    <div className="grid gap-6 lg:grid-cols-[20rem_1fr] lg:items-start">
      <div className="space-y-6">
        <Panel id="photo-title" title="Photo de profil">
          <div className="mb-6 flex items-center gap-4">
            <Avatar name={profile.alias} src={profile.avatarUrl} size={80} />
            <div>
              <p className="font-display text-lg font-semibold">{profile.alias}</p>
              <p className="text-sm text-muted">
                Membre depuis le{" "}
                <time dateTime={profile.registeredAt}>{formatDate(profile.registeredAt)}</time>
              </p>
            </div>
          </div>
          <AvatarForm action={updateHeroAvatarAction} />
        </Panel>

        <Panel
          id="identity-title"
          title="Informations personnelles"
          description="Pour les modifier, contactez la modération."
        >
          <dl className="space-y-3">
            {[
              ["Pseudo", profile.alias],
              ["Pouvoir", profile.power],
              ["Civilité", civility],
              ["Nom", formatFullName(profile.lastName, profile.firstName)],
              ["Date de naissance", profile.birthDate && formatDate(profile.birthDate)],
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
          <ContactForm contact={profile} action={updateHeroContactAction} withComplement={false} />
        </Panel>

        <Panel id="email-title" title="Adresse email">
          <p className="text-sm text-muted">Adresse actuelle</p>
          <p className="mb-4 font-medium break-all">{profile.email}</p>
          {profile.pendingEmail && (
            <p className="mb-6 rounded-xl border border-rating/40 bg-rating/10 px-4 py-3 text-sm">
              <span className="font-semibold text-rating">En attente de modération :</span>{" "}
              <span className="break-all">{profile.pendingEmail}</span>
            </p>
          )}
          <EmailChangeForm action={requestHeroEmailChangeAction} />
        </Panel>
      </div>
    </div>
  );
}
