import type { ReactNode } from "react";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";

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
    title: "Le citoyen signale",
    description:
      "Décrivez l'incident, le lieu et le vilain impliqué depuis votre téléphone. Cela prend moins d'une minute.",
    icon: (
      <svg {...iconProps}>
        <path d="M12 9v4M12 17h.01" />
        <path d="M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0Z" />
      </svg>
    ),
  },
  {
    title: "Un super-héros intervient",
    description:
      "Le héros disponible le plus proche accepte la mission et vous suivez son intervention en temps réel.",
    icon: (
      <svg {...iconProps}>
        <path d="M13 2 3 14h9l-1 8 10-12h-9l1-8Z" />
      </svg>
    ),
  },
  {
    title: "Le citoyen note l'intervention",
    description:
      "Une fois l'incident résolu, évaluez le héros. Les notes garantissent la qualité des interventions.",
    icon: (
      <svg {...iconProps}>
        <path d="m12 2 3.1 6.3 6.9 1-5 4.9 1.2 6.8L12 17.8 5.8 21l1.2-6.8-5-4.9 6.9-1L12 2Z" />
      </svg>
    ),
  },
];

export function HowItWorksSection() {
  return (
    <section
      id="comment-ca-marche"
      aria-labelledby="how-title"
      className="scroll-mt-16 border-y border-border bg-surface py-16 sm:py-20"
    >
      <Container>
        <SectionHeading
          id="how-title"
          eyebrow="Comment ça marche"
          title="Trois étapes pour ramener le calme"
          className="mb-10"
        />
        <ol className="grid gap-6 md:grid-cols-3">
          {steps.map((step, index) => (
            <li
              key={step.title}
              className="relative rounded-2xl border border-border bg-background p-6"
            >
              <div className="flex items-center gap-4">
                <span className="inline-flex size-12 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  {step.icon}
                </span>
                <span aria-hidden="true" className="font-display text-4xl font-bold text-border">
                  {String(index + 1).padStart(2, "0")}
                </span>
              </div>
              <h3 className="mt-5 font-display text-lg font-semibold">
                <span className="sr-only">Étape {index + 1} : </span>
                {step.title}
              </h3>
              <p className="mt-2 text-sm text-muted">{step.description}</p>
            </li>
          ))}
        </ol>
      </Container>
    </section>
  );
}
