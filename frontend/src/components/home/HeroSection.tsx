import Image from "next/image";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { Container } from "@/components/ui/Container";
import { Typewriter } from "@/components/ui/Typewriter";
import { routes } from "@/lib/routes";

const requiredItems = [
  "Nature de l'incident",
  "Lieu des faits",
  "Identité du super-vilain (si connue)",
];

/** Affiche encadrée : décorative, masquée sur petit écran. */
function Poster() {
  return (
    <figure className="panel rivets paper relative mx-auto hidden w-full max-w-xs rotate-1 p-3 pb-2 md:block lg:max-w-sm">
      <Image
        src="/ucnr2b18wx471.jpg"
        alt=""
        width={1242}
        height={1652}
        priority
        sizes="(min-width: 1024px) 384px, 320px"
        className="h-auto w-full border-2 border-background"
      />
      <figcaption
        aria-hidden="true"
        className="mt-2 ps-2.5 font-mono text-[0.65rem] tracking-widest text-background/75 uppercase"
      >
        Affiche officielle n° 7-B — ne pas détériorer
      </figcaption>
      <span
        aria-hidden="true"
        className="stamp absolute top-8 -left-3 bg-paper/70 text-lg lg:-left-6 lg:text-xl"
      >
        Priorité publique
      </span>
    </figure>
  );
}

export function HeroSection() {
  return (
    <section
      aria-labelledby="hero-title"
      className="relative isolate overflow-hidden border-b-4 border-[#0d0906]"
    >
      <div aria-hidden="true" className="bg-tiles absolute inset-0 -z-10 opacity-60" />
      <div
        aria-hidden="true"
        className="absolute inset-0 -z-10 bg-[radial-gradient(ellipse_at_left,var(--color-background)_30%,transparent_85%)]"
      />
      <Container className="grid items-center gap-12 py-12 sm:py-16 lg:grid-cols-[1.2fr_1fr] lg:py-24">
        <div>
          <p className="mb-5 inline-flex flex-wrap items-center gap-x-3 gap-y-1 border-2 border-border bg-background px-3 py-1.5 font-mono text-xs tracking-widest text-muted uppercase">
            <span>Formulaire n° 7-B</span>
            <span aria-hidden="true" className="text-border">
              |
            </span>
            <span className="inline-flex items-center gap-2">
              <span aria-hidden="true" className="led" />
              Guichet ouvert 24 h/24
            </span>
          </p>

          <h1
            id="hero-title"
            className="font-display text-3xl leading-tight font-bold tracking-widest text-balance uppercase sm:text-4xl lg:text-5xl"
          >
            <Typewriter text="Un super-vilain sévit ? Un agent est dépêché." />
          </h1>

          <p className="mt-6 max-w-xl text-lg text-muted text-pretty">
            Veuillez déclarer tout incident impliquant un super-vilain. Votre signalement sera
            traité dans l&apos;ordre de priorité par l&apos;agent disponible le plus proche.
          </p>

          <div className="mt-6 max-w-xl border-2 border-border bg-background/80 p-4">
            <h2 className="font-mono text-xs font-medium tracking-widest text-accent uppercase">
              Pièces à fournir
            </h2>
            <ul className="mt-3 space-y-2 text-sm">
              {requiredItems.map((item, index) => (
                <li key={item} className="flex items-center gap-3">
                  <span aria-hidden="true" className="size-4 shrink-0 border-2 border-muted" />
                  <span className="font-mono text-xs text-muted">{`${index + 1}.`}</span>
                  {item}
                </li>
              ))}
            </ul>
          </div>

          <div className="mt-8 flex flex-col gap-4 sm:flex-row sm:items-center">
            <ButtonLink href={routes.signup} variant="console" size="lg" className="gap-3">
              <span aria-hidden="true" className="led" />
              Signaler un incident
            </ButtonLink>
            <ButtonLink href="#comment-ca-marche" variant="secondary" size="lg">
              Consulter la procédure
            </ButtonLink>
          </div>

          <p className="mt-6 max-w-xl text-xs text-muted">
            Toute déclaration abusive est passible d&apos;un entretien avec le Service (art. 7-B,
            al. 3). Univers fictif : tous les personnages sont imaginaires.
          </p>
        </div>

        <Poster />
      </Container>
    </section>
  );
}
