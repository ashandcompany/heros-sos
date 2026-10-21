import type { Metadata } from "next";
import Link from "next/link";
import { ContactForm } from "@/components/contact/ContactForm";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Contact",
  description:
    "Écrivez au Bureau Heros-SOS : question, problème de compte ou données personnelles.",
};

const desk = [
  { label: "Délai de réponse", value: "5 jours ouvrés" },
  { label: "Réponse par", value: "Email" },
  { label: "Guichet", value: "Correspondance" },
];

export default function ContactPage() {
  return (
    <Container className="py-10 sm:py-14">
      <header className="max-w-3xl">
        <p className="mb-5 inline-flex flex-wrap items-center gap-x-3 gap-y-1 border-2 border-edge bg-background px-3 py-1.5 font-mono text-xs tracking-widest text-muted uppercase">
          <span>Formulaire n° 12-C</span>
          <span aria-hidden="true" className="text-border">
            |
          </span>
          <span className="inline-flex items-center gap-2">
            <span aria-hidden="true" className="led" />
            Guichet de la correspondance
          </span>
        </p>
        <h1 className="font-display text-4xl leading-[1.05] font-black tracking-tight text-balance uppercase sm:text-5xl">
          Écrire au Bureau
        </h1>
        <p className="mt-4 text-lg text-pretty text-muted">
          Une question sur le service, un souci de compte ou une demande sur vos données ?
          Remplissez le formulaire ci-dessous : un agent du Bureau vous répond par email.
        </p>
        <div aria-hidden="true" className="bar-tricolor mt-6 w-48" />
      </header>

      <div className="mt-12 grid items-start gap-10 lg:grid-cols-[1.6fr_1fr]">
        <section aria-labelledby="contact-form-title" className="panel rivets p-6 sm:p-8">
          <h2
            id="contact-form-title"
            className="mb-6 border-b-2 border-border pb-3 font-mono text-xs font-medium tracking-widest text-accent uppercase"
          >
            Votre message
          </h2>
          <ContactForm />
        </section>

        <aside aria-label="Informations pratiques" className="space-y-8">
          <div className="border-4 border-brick bg-background p-5 shadow-[4px_4px_0_0_var(--color-shadow)]">
            <p className="flex items-center gap-2 font-mono text-xs font-semibold tracking-widest text-brick uppercase">
              <span aria-hidden="true" className="led bg-brick shadow-none" />
              Urgence en cours ?
            </p>
            <p className="mt-3 text-sm text-pretty">
              Ce formulaire n&apos;est pas surveillé en continu. Si un super-vilain sévit, déclarez
              l&apos;incident pour qu&apos;un agent soit dépêché.
            </p>
            <ButtonLink href={routes.profileIncidents} variant="danger" className="mt-4 w-full">
              Signaler un incident
            </ButtonLink>
          </div>

          <div className="crt">
            <div className="crt__screen">
              <dl className="space-y-4">
                {desk.map((item) => (
                  <div key={item.label}>
                    <dt className="font-mono text-[0.65rem] tracking-widest text-muted uppercase">
                      {item.label}
                    </dt>
                    <dd className="digital mt-1 text-lg">{item.value}</dd>
                  </div>
                ))}
              </dl>
            </div>
          </div>

          <p className="text-sm text-pretty text-muted">
            Vous cherchez une réponse rapide ? Consultez les{" "}
            <Link href={routes.faq} className="text-accent underline">
              questions fréquentes
            </Link>
            .
          </p>
        </aside>
      </div>
    </Container>
  );
}
