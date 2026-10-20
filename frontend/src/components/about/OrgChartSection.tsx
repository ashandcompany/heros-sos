import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";

interface Service {
  name: string;
  role: string;
  description: string;
}

const bureau: Service = {
  name: "Le Bureau",
  role: "Modération",
  description:
    "Vérifie les déclarations, valide les agents, veille au respect du règlement et tient les archives.",
};

const branches: Service[] = [
  {
    name: "Les citoyens",
    role: "Déclarent et évaluent",
    description:
      "Signalent les incidents, suivent leur traitement et notent l'agent une fois le dossier clos.",
  },
  {
    name: "Les super-héros",
    role: "Interviennent",
    description:
      "Prennent les dossiers près de chez eux, rendent compte de l'issue et soignent leur réputation.",
  },
];

function ServiceCard({ service, featured = false }: { service: Service; featured?: boolean }) {
  return (
    <div
      className={featured ? "panel rivets bg-primary! p-6 text-primary-foreground" : "panel p-6"}
    >
      <p className="font-mono text-xs tracking-widest uppercase opacity-80">{service.role}</p>
      <h3 className="mt-1 font-display text-2xl font-black tracking-wider uppercase">
        {service.name}
      </h3>
      <p className="mt-3 text-sm text-pretty">{service.description}</p>
    </div>
  );
}

/** Organigramme : le Bureau au sommet, citoyens et super-héros en dessous. */
export function OrgChartSection() {
  return (
    <section
      aria-labelledby="orgchart-title"
      className="bg-wood border-y-4 border-edge py-14 sm:py-20"
    >
      <Container>
        <SectionHeading
          id="orgchart-title"
          section="04"
          eyebrow="Organigramme"
          title="Qui fait quoi au Bureau"
          description="Trois services, un seul circuit. Le dossier descend du Bureau vers l'agent, la note remonte du citoyen vers le registre."
          className="mb-12"
        />

        <div className="mx-auto max-w-4xl">
          <div className="mx-auto max-w-md">
            <ServiceCard service={bureau} featured />
          </div>

          {/* Filets de l'organigramme : tronc, barre horizontale, puis une descente par branche */}
          <div aria-hidden="true" className="mx-auto h-10 w-1 bg-edge" />
          <div
            aria-hidden="true"
            className="mx-auto hidden h-1 w-[calc(50%+1rem)] bg-edge sm:block"
          />

          <ul className="grid gap-10 sm:grid-cols-2 sm:gap-8">
            {branches.map((branch) => (
              <li key={branch.name}>
                <div aria-hidden="true" className="mx-auto hidden h-8 w-1 bg-edge sm:block" />
                <ServiceCard service={branch} />
              </li>
            ))}
          </ul>
        </div>
      </Container>
    </section>
  );
}
