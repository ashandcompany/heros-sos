import type { ReactNode } from "react";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { WallPoster } from "@/components/ui/WallPoster";

interface Step {
  title: string;
  description: string;
  icon: ReactNode;
}

const iconProps = {
  viewBox: "0 0 24 24",
  "aria-hidden": true,
  className: "size-6 fill-none stroke-current",
  strokeWidth: 2,
  strokeLinecap: "round",
  strokeLinejoin: "round",
} as const;

const steps: Step[] = [
  {
    title: "Dépôt du signalement",
    description:
      "Veuillez décrire l'incident, le lieu et le vilain impliqué. La formalité dure moins d'une minute ; aucun timbre n'est exigé.",
    icon: (
      <svg {...iconProps}>
        <path d="M12 9v4M12 17h.01" />
        <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
      </svg>
    ),
  },
  {
    title: "Affectation d'un agent",
    description:
      "L'agent disponible le plus proche prend le dossier en charge. Vous suivez son intervention en temps réel.",
    icon: (
      <svg {...iconProps}>
        <path d="M13 2 3 14h9l-1 8 10-12h-9l1-8Z" />
      </svg>
    ),
  },
  {
    title: "Évaluation de l'intervention",
    description:
      "Une fois le dossier clos, veuillez noter l'agent. Les évaluations sont lues, classées puis archivées.",
    icon: (
      <svg {...iconProps}>
        <path d="m12 2 3.1 6.3 6.9 1-5 4.9 1.2 6.8L12 17.8 5.8 21l1.2-6.8-5-4.9 6.9-1L12 2Z" />
      </svg>
    ),
  },
];

/** Flèche épaisse reliant deux guichets : verticale sur mobile, horizontale dès md. */
function ProcedureArrow() {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 48 48"
      className="absolute top-full left-1/2 h-10 w-12 -translate-x-1/2 text-accent md:top-1/2 md:left-full md:h-12 md:w-12 md:-translate-y-1/2 md:translate-x-0"
    >
      <g className="md:hidden">
        <path d="M24 0v28" stroke="currentColor" strokeWidth="6" />
        <path d="M10 26h28L24 44Z" fill="currentColor" />
      </g>
      <g className="hidden md:block">
        <path d="M0 24h30" stroke="currentColor" strokeWidth="6" />
        <path d="M28 10v28l18-14Z" fill="currentColor" />
      </g>
    </svg>
  );
}

export function HowItWorksSection() {
  return (
    <section
      id="comment-ca-marche"
      aria-labelledby="how-title"
      className="bg-wood scroll-mt-28 border-y-4 border-edge py-14 sm:py-20"
    >
      <Container>
        <div className="mb-12 flex items-start justify-between gap-10">
          <SectionHeading
            id="how-title"
            section="02"
            eyebrow="Procédure"
            title="Trois guichets pour ramener le calme"
            description="Votre dossier circule de guichet en guichet. Veuillez patienter à chaque étape ; la file d'attente est, dans l'ensemble, courte."
            className="self-center"
          />
          <WallPoster
            src="/affiche2.png"
            width={484}
            height={646}
            caption="Affichage obligatoire — salle d'attente"
            className="rotate-2"
          />
        </div>
        <ol className="grid gap-14 md:grid-cols-3 md:gap-12">
          {steps.map((step, index) => (
            <li key={step.title} className="panel rivets paper relative p-6 pt-5">
              <div className="flex items-center justify-between gap-4 border-b-2 border-background pb-3">
                <span
                  aria-hidden="true"
                  className="font-mono text-xs font-medium tracking-widest uppercase"
                >
                  Guichet n° {String(index + 1).padStart(2, "0")}
                </span>
                <span className="inline-flex size-11 items-center justify-center border-2 border-background bg-primary text-primary-foreground">
                  {step.icon}
                </span>
              </div>
              <h3 className="mt-5 font-display text-lg font-bold tracking-wider text-balance uppercase">
                <span className="sr-only">Étape {index + 1} : </span>
                {step.title}
              </h3>
              <p className="mt-2 text-sm text-pretty text-ink/85">{step.description}</p>
              {index < steps.length - 1 && <ProcedureArrow />}
            </li>
          ))}
        </ol>
      </Container>
    </section>
  );
}
