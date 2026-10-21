import { Container } from "@/components/ui/Container";
import { Typewriter } from "@/components/ui/Typewriter";
import { AtomSeal } from "./AtomSeal";

/** En-tête de la page À propos : titre machine à écrire et sceau atomique du Bureau. */
export function AboutHero() {
  return (
    <section
      aria-labelledby="about-title"
      className="relative isolate overflow-hidden border-b-4 border-edge"
    >
      <div aria-hidden="true" className="bg-tiles absolute inset-0 -z-10 opacity-70" />
      <div
        aria-hidden="true"
        className="starburst absolute -bottom-48 -left-48 -z-10 size-[36rem] opacity-[0.12] motion-safe:animate-[spin_140s_linear_infinite]"
      />
      <div
        aria-hidden="true"
        className="absolute inset-0 -z-10 bg-[radial-gradient(ellipse_at_left,var(--color-background)_30%,transparent_85%)]"
      />

      <Container className="grid items-center gap-12 py-12 sm:py-16 lg:grid-cols-[1.3fr_1fr] lg:py-24">
        <div>
          <p className="mb-5 inline-flex flex-wrap items-center gap-x-3 gap-y-1 border-2 border-edge bg-background px-3 py-1.5 font-mono text-xs tracking-widest text-muted uppercase">
            <span>Brochure institutionnelle</span>
            <span aria-hidden="true" className="text-border">
              |
            </span>
            <span>Édition 1957 — aujourd&apos;hui</span>
          </p>

          <h1
            id="about-title"
            className="font-display text-4xl leading-[1.05] font-black tracking-tight text-balance uppercase sm:text-5xl lg:text-6xl"
          >
            <Typewriter text="Depuis 1957, le Bureau envoie les héros." />
          </h1>

          <p className="mt-6 max-w-xl text-lg text-pretty text-muted">
            Heros-SOS est le guichet unique entre les citoyens qui subissent les méfaits des
            super-vilains et les super-héros qui y mettent fin. Un standard téléphonique hier, un
            formulaire en ligne aujourd&apos;hui : la mission, elle, n&apos;a pas changé.
          </p>

          <div aria-hidden="true" className="bar-tricolor mt-8 w-48" />

          <p className="mt-6 max-w-xl text-xs text-muted">
            Univers fictif : le Bureau, ses archives et tous les personnages sont imaginaires.
          </p>
        </div>

        <figure className="mx-auto w-full max-w-72 sm:max-w-80 lg:max-w-96">
          <AtomSeal className="w-full drop-shadow-[6px_6px_0_var(--color-shadow)]" />
          <figcaption
            aria-hidden="true"
            className="mt-4 text-center font-mono text-[0.65rem] tracking-widest text-muted uppercase"
          >
            Sceau officiel — reproduction interdite
          </figcaption>
        </figure>
      </Container>
    </section>
  );
}
