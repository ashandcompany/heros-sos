import Link from "next/link";
import type { ReactNode } from "react";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { routes } from "@/lib/routes";

interface Principle {
  title: string;
  description: ReactNode;
  /** Pastille de couleur de l'article, reprise de la bande tricolore. */
  tone: string;
}

const principles: Principle[] = [
  {
    title: "Le citoyen d'abord",
    description:
      "Chaque déclaration est lue par un agent du Bureau. Aucun incident n'est trop petit pour être pris au sérieux.",
    tone: "bg-primary",
  },
  {
    title: "L'agent le plus proche",
    description:
      "Pas de favoritisme ni de tête d'affiche : le dossier revient au super-héros disponible le plus près des faits.",
    tone: "bg-mustard",
  },
  {
    title: "Chaque intervention est notée",
    description:
      "Les citoyens évaluent les agents après chaque intervention. Le registre est public et la réputation se mérite.",
    tone: "bg-accent",
  },
  {
    title: "Vos données sous scellés",
    description: (
      <>
        Nous ne collectons que le nécessaire et ne vendons rien. Le détail figure dans notre{" "}
        <Link href={routes.privacy} className="font-semibold underline">
          politique de confidentialité
        </Link>
        .
      </>
    ),
    tone: "bg-sage",
  },
];

/** Les quatre articles du règlement intérieur, sur fiches crème. */
export function PrinciplesSection() {
  return (
    <section
      aria-labelledby="principles-title"
      className="bg-wood border-y-4 border-edge py-14 sm:py-20"
    >
      <Container>
        <SectionHeading
          id="principles-title"
          section="02"
          eyebrow="Règlement intérieur"
          title="Quatre articles, aucune dérogation"
          description="Affiché dans chaque bureau, relu à chaque prise de poste. Les agents qui l'oublient sont priés de le recopier à la main."
          className="mb-12"
        />
        <ol className="grid gap-8 sm:grid-cols-2 lg:grid-cols-4">
          {principles.map((principle, index) => (
            <li key={principle.title} className="panel paper relative flex flex-col p-6 pt-7">
              <div className="flex items-center justify-between gap-4 border-b-2 border-ink/20 pb-3">
                <span className="font-mono text-xs font-medium tracking-widest uppercase">
                  Article {index === 0 ? "1er" : index + 1}
                </span>
                <span
                  aria-hidden="true"
                  className={`size-6 rounded-full border-2 border-ink ${principle.tone}`}
                />
              </div>
              <h3 className="mt-5 font-display text-lg font-bold tracking-wider text-balance uppercase">
                {principle.title}
              </h3>
              <p className="mt-2 text-sm text-pretty text-ink/85">{principle.description}</p>
            </li>
          ))}
        </ol>
      </Container>
    </section>
  );
}
