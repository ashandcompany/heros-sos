import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { cn } from "@/components/ui/cn";

interface Milestone {
  year: string;
  title: string;
  description: string;
}

const milestones: Milestone[] = [
  {
    year: "1957",
    title: "Ouverture du Bureau",
    description:
      "Un standard à quatre lignes, un tableau en liège et une carte de France punaisée. Le premier appel concerne un chat coincé dans un rayon tracteur.",
  },
  {
    year: "1962",
    title: "Le registre des agents",
    description:
      "Douze super-héros signent le registre, sur fiches cartonnées classées par ordre alphabétique de super-pouvoir.",
  },
  {
    year: "1969",
    title: "La ligne SOS",
    description:
      "Le célèbre numéro à cadran entre dans les foyers. Les opératrices transmettent les alertes par pneumatique.",
  },
  {
    year: "1973",
    title: "Naissance du formulaire 7-B",
    description:
      "Trois feuillets carbone, un seul objectif : décrire l'incident, le lieu et le vilain. Il n'a presque pas changé depuis.",
  },
  {
    year: "1976",
    title: "La note des citoyens",
    description:
      "Les victimes évaluent les agents sur carte perforée. Le classement est affiché chaque lundi dans le hall.",
  },
  {
    year: "Auj.",
    title: "Heros-SOS",
    description:
      "Le guichet passe en ligne : déclaration en une minute, carte des incidents en temps réel et réputation des agents publique.",
  },
];

// Pastilles en rotation sur les trois couleurs de la bande tricolore.
const dotTones = ["bg-primary", "bg-mustard", "bg-accent"];

/** Frise chronologique du Bureau : titre fixé à gauche sur grand écran, jalons à droite. */
export function TimelineSection() {
  return (
    <section aria-labelledby="timeline-title" className="py-14 sm:py-20">
      <Container className="grid gap-12 lg:grid-cols-[1fr_1.6fr]">
        <div className="lg:sticky lg:top-28 lg:self-start">
          <SectionHeading
            id="timeline-title"
            section="03"
            eyebrow="Archives"
            title="Petite histoire du Bureau"
            description="Extraits des archives, sortis des cartons pour l'occasion. Merci de ne pas plier les coins."
          />
        </div>

        <ol className="relative ml-3 border-l-4 border-edge">
          {milestones.map((milestone, index) => (
            <li key={milestone.year} className="relative pb-10 pl-10 last:pb-0">
              <span
                aria-hidden="true"
                className={cn(
                  "absolute top-3 -left-[0.9rem] size-6 rounded-full border-4 border-edge",
                  dotTones[index % dotTones.length],
                )}
              />
              <p className="font-display text-5xl leading-none font-black tracking-tight text-transparent [-webkit-text-stroke:2px_var(--color-foreground)] sm:text-6xl">
                {milestone.year === "Auj." ? (
                  <abbr title="Aujourd'hui" className="no-underline">
                    Auj.
                  </abbr>
                ) : (
                  milestone.year
                )}
              </p>
              <div className="panel mt-4 p-5">
                <h3 className="font-display text-lg font-bold tracking-wider uppercase">
                  {milestone.title}
                </h3>
                <p className="mt-2 text-sm text-pretty text-muted">{milestone.description}</p>
              </div>
            </li>
          ))}
        </ol>
      </Container>
    </section>
  );
}
