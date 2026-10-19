import { ButtonLink } from "@/components/ui/ButtonLink";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

function SignalIllustration() {
  return (
    <div
      aria-hidden="true"
      className="relative mx-auto hidden aspect-square w-full max-w-sm lg:block"
    >
      <div className="absolute inset-0 rounded-full border border-border/60" />
      <div className="absolute inset-[12%] rounded-full border border-border/70" />
      <div className="absolute inset-[24%] rounded-full border border-accent/30" />
      <div className="absolute inset-[36%] rounded-full bg-primary/15 motion-safe:animate-pulse" />
      <svg viewBox="0 0 64 64" className="absolute inset-[38%]">
        <path
          d="M32 4 8 14v17c0 15 10.4 26.2 24 29 13.6-2.8 24-14 24-29V14L32 4Z"
          className="fill-surface-raised stroke-primary"
          strokeWidth="3"
          strokeLinejoin="round"
        />
        <path
          d="M16 34h8l4-10 8 20 4-10h8"
          className="stroke-accent"
          fill="none"
          strokeWidth="3"
          strokeLinecap="round"
          strokeLinejoin="round"
        />
      </svg>
      <span className="absolute top-[18%] right-[14%] size-3 rounded-full bg-accent shadow-[0_0_16px] shadow-accent" />
      <span className="absolute bottom-[22%] left-[10%] size-2 rounded-full bg-primary shadow-[0_0_16px] shadow-primary" />
      <span className="absolute right-[30%] bottom-[8%] size-2 rounded-full bg-success shadow-[0_0_12px] shadow-success" />
    </div>
  );
}

export function HeroSection() {
  return (
    <section aria-labelledby="hero-title" className="relative isolate overflow-hidden">
      <div
        aria-hidden="true"
        className="absolute inset-0 -z-10 bg-[radial-gradient(ellipse_at_top_right,color-mix(in_oklab,var(--color-primary)_18%,transparent),transparent_55%),radial-gradient(ellipse_at_bottom_left,color-mix(in_oklab,var(--color-accent)_12%,transparent),transparent_50%)]"
      />
      <Container className="grid items-center gap-12 py-16 sm:py-20 lg:grid-cols-[1.2fr_1fr] lg:py-28">
        <div>
          <p className="mb-4 inline-flex items-center gap-2 rounded-full border border-border bg-surface px-3 py-1 text-sm text-muted">
            <span aria-hidden="true" className="size-2 rounded-full bg-success" />
            Des héros disponibles 24 h/24
          </p>
          <h1
            id="hero-title"
            className="font-display text-4xl leading-tight font-bold tracking-tight text-balance sm:text-5xl lg:text-6xl"
          >
            Un super-vilain frappe&nbsp;? <span className="text-primary">Un héros répond.</span>
          </h1>
          <p className="mt-6 max-w-xl text-lg text-muted text-pretty">
            Heros-SOS met en relation les citoyens et les super-héros. Signalez un vol, un
            cambriolage ou une prise d&apos;otages en quelques secondes depuis votre téléphone : le
            héros le plus proche prend le relais.
          </p>
          <div className="mt-8 flex flex-col gap-3 sm:flex-row">
            <ButtonLink href={routes.signup} size="lg">
              Signaler un incident
            </ButtonLink>
            <ButtonLink href="#comment-ca-marche" variant="secondary" size="lg">
              Comment ça marche&nbsp;?
            </ButtonLink>
          </div>
        </div>
        <SignalIllustration />
      </Container>
    </section>
  );
}
