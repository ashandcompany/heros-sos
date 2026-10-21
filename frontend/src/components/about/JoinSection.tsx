import { ButtonLink } from "@/components/ui/ButtonLink";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

/** Appel final : ouvrir un dossier ou consulter la carte des incidents. */
export function JoinSection() {
  return (
    <section
      aria-labelledby="join-title"
      className="relative isolate overflow-hidden py-16 sm:py-24"
    >
      <div
        aria-hidden="true"
        className="starburst absolute top-1/2 left-1/2 -z-10 size-[44rem] -translate-x-1/2 -translate-y-1/2 opacity-[0.1]"
      />
      <Container>
        <div className="panel rivets paper relative mx-auto max-w-3xl p-8 text-center sm:p-12">
          <span
            aria-hidden="true"
            className="stamp absolute -top-4 -right-2 bg-paper text-sm sm:-right-6 sm:text-base"
          >
            Guichet ouvert
          </span>
          <p className="font-mono text-xs tracking-widest uppercase">Section 05 — Adhésion</p>
          <h2
            id="join-title"
            className="mt-3 font-display text-3xl font-black tracking-[0.04em] text-balance uppercase sm:text-4xl"
          >
            Votre dossier commence ici
          </h2>
          <div aria-hidden="true" className="bar-tricolor mx-auto mt-5 w-32" />
          <p className="mx-auto mt-5 max-w-xl text-pretty text-ink/85">
            Créez votre compte pour déclarer un incident et suivre son traitement, ou consultez la
            carte pour savoir où les super-vilains ont frappé récemment.
          </p>
          <div className="mt-8 flex flex-col justify-center gap-4 sm:flex-row">
            <ButtonLink href={routes.signup} variant="console" size="lg" className="gap-3">
              <span aria-hidden="true" className="led" />
              Ouvrir un dossier
            </ButtonLink>
            <ButtonLink href={routes.map} variant="secondary" size="lg">
              Voir la carte
            </ButtonLink>
          </div>
        </div>
      </Container>
    </section>
  );
}
