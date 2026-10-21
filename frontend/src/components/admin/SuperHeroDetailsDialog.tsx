import type { ReactNode } from "react";
import { Avatar } from "@/components/ui/Avatar";
import { RatingStars } from "@/components/ui/RatingStars";
import { countries } from "@/lib/countries";
import { formatDate, formatInteger, formatPhone, phoneHref } from "@/lib/format";
import { civilityOptions } from "@/lib/validation/signup";
import type { SuperHeroProfile } from "@/types";

export const heroDialogId = (heroId: string) => `fiche-${heroId}`;

function Unknown() {
  return <span className="text-muted italic">Non communiqué</span>;
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="border-t border-border pt-5">
      <h3 className="mb-3 text-sm font-semibold tracking-wider text-accent uppercase">{title}</h3>
      <dl className="grid gap-x-6 gap-y-3 sm:grid-cols-2">{children}</dl>
    </section>
  );
}

function Item({ label, children, wide }: { label: string; children: ReactNode; wide?: boolean }) {
  return (
    <div className={wide ? "sm:col-span-2" : undefined}>
      <dt className="text-sm text-muted">{label}</dt>
      <dd className="mt-0.5 break-words">{children}</dd>
    </div>
  );
}

/** Fiche complète d'un super-héros dans un <dialog> natif (fermeture sans JavaScript). */
export function SuperHeroDetailsDialog({ hero }: { hero: SuperHeroProfile }) {
  const titleId = `${heroDialogId(hero.id)}-titre`;
  const civility = civilityOptions.find((option) => option.value === hero.civility)?.label;
  const country = countries.find((c) => c.code === hero.address?.country)?.name;

  return (
    <dialog
      id={heroDialogId(hero.id)}
      aria-labelledby={titleId}
      className="m-auto max-h-[calc(100dvh-2rem)] w-[calc(100%-2rem)] max-w-2xl overflow-y-auto border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-0 text-foreground backdrop:bg-black/70 backdrop:backdrop-blur-sm"
    >
      <div className="flex items-start gap-4 p-6 sm:p-8">
        <Avatar name={hero.alias} src={hero.avatarUrl} size={64} />
        <div className="min-w-0 flex-1">
          <h2 id={titleId} className="font-display text-2xl font-bold">
            {hero.alias}
          </h2>
          <p className="mt-1 text-sm text-muted">{hero.power}</p>
        </div>
        <form method="dialog">
          <button
            type="submit"
            aria-label="Fermer la fiche"
            className="-mt-2 -mr-2 inline-flex size-11 items-center justify-center rounded-lg text-muted hover:bg-surface-raised hover:text-foreground"
          >
            <svg viewBox="0 0 24 24" aria-hidden="true" className="size-5 fill-none stroke-current">
              <path d="M6 6l12 12M18 6 6 18" strokeWidth="2" strokeLinecap="round" />
            </svg>
          </button>
        </form>
      </div>

      <div className="space-y-5 px-6 pb-6 sm:px-8 sm:pb-8">
        <Section title="Identité civile">
          <Item label="Civilité">{civility ?? <Unknown />}</Item>
          <Item label="Date de naissance">
            {hero.birthDate ? (
              <time dateTime={hero.birthDate}>{formatDate(hero.birthDate)}</time>
            ) : (
              <Unknown />
            )}
          </Item>
          <Item label="Nom">{hero.lastName ?? <Unknown />}</Item>
          <Item label="Prénom">{hero.firstName ?? <Unknown />}</Item>
        </Section>

        <Section title="Coordonnées">
          <Item label="Adresse email">
            <a href={`mailto:${hero.email}`} className="text-accent hover:underline">
              {hero.email}
            </a>
          </Item>
          <Item label="Numéro de portable">
            <a href={phoneHref(hero.mobilePhone)} className="text-accent hover:underline">
              {formatPhone(hero.mobilePhone)}
            </a>
          </Item>
          <Item label="Adresse postale" wide>
            {hero.address ? (
              <address className="not-italic">
                {hero.address.street}
                <br />
                {hero.address.postalCode} {hero.address.city}
                {country && (
                  <>
                    <br />
                    {country}
                  </>
                )}
              </address>
            ) : (
              <Unknown />
            )}
          </Item>
        </Section>

        <Section title="Activité">
          <Item label="Note moyenne">
            <RatingStars rating={hero.averageRating} />
          </Item>
          <Item label="Interventions">{formatInteger(hero.interventionsCount)}</Item>
          <Item label="Inscrit le">
            <time dateTime={hero.registeredAt}>{formatDate(hero.registeredAt)}</time>
          </Item>
          <Item label="Dernière intervention">
            {hero.lastInterventionAt ? (
              <time dateTime={hero.lastInterventionAt}>{formatDate(hero.lastInterventionAt)}</time>
            ) : (
              <span className="text-muted italic">Aucune</span>
            )}
          </Item>
        </Section>
      </div>
    </dialog>
  );
}
