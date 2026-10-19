import { Suspense, type ReactNode } from "react";
import { Carousel } from "@/components/ui/Carousel";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { getLatestResolvedIncidents } from "@/lib/api";
import { IncidentCard } from "./IncidentCard";

/** Cadre d'écran cathodique : courbure, lignes de balayage et scintillement au chargement. */
function CrtFrame({ children }: { children: ReactNode }) {
  return (
    <div className="crt">
      <div className="crt__screen">
        <p
          aria-hidden="true"
          className="mb-4 flex items-center justify-between font-mono text-xs tracking-widest text-terminal uppercase"
        >
          <span>Terminal 07 — Archives</span>
          <span className="inline-flex items-center gap-2">
            <span className="led" />
            En ligne
          </span>
        </p>
        {children}
      </div>
    </div>
  );
}

async function LatestIncidentsCarousel() {
  const incidents = await getLatestResolvedIncidents();

  if (incidents.length === 0) {
    return <p className="text-muted">Aucun dossier clos pour le moment.</p>;
  }

  return (
    <Carousel
      label="Derniers incidents résolus"
      previousLabel="Dossiers précédents"
      nextLabel="Dossiers suivants"
    >
      {incidents.map((incident) => (
        <IncidentCard key={incident.id} incident={incident} />
      ))}
    </Carousel>
  );
}

function CarouselSkeleton() {
  return (
    <div className="flex gap-4 overflow-hidden pt-7" aria-hidden="true">
      {Array.from({ length: 3 }, (_, index) => (
        <div
          key={index}
          className="paper h-72 w-[85%] shrink-0 border-2 border-edge opacity-40 motion-safe:animate-pulse sm:w-[calc((100%-1rem)/2)] lg:w-[calc((100%-2rem)/3)]"
        />
      ))}
    </div>
  );
}

export function LatestIncidentsSection() {
  return (
    <section aria-labelledby="incidents-title" className="py-14 sm:py-20">
      <Container>
        <SectionHeading
          id="incidents-title"
          section="01"
          eyebrow="Dossiers clos"
          title="Derniers incidents traités avec succès"
          description="Les dossiers ci-dessous ont été instruits, puis classés par nos agents. Aucune suite n'est requise de votre part."
          className="mb-8"
        />
        <CrtFrame>
          <Suspense fallback={<CarouselSkeleton />}>
            <LatestIncidentsCarousel />
          </Suspense>
        </CrtFrame>
      </Container>
    </section>
  );
}
