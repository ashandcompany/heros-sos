import { Suspense } from "react";
import { Carousel } from "@/components/ui/Carousel";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { getLatestResolvedIncidents } from "@/lib/api";
import { IncidentCard } from "./IncidentCard";

async function LatestIncidentsCarousel() {
  const incidents = await getLatestResolvedIncidents();

  if (incidents.length === 0) {
    return <p className="text-muted">Aucun incident résolu pour le moment.</p>;
  }

  return (
    <Carousel
      label="Derniers incidents résolus"
      previousLabel="Incidents précédents"
      nextLabel="Incidents suivants"
    >
      {incidents.map((incident) => (
        <IncidentCard key={incident.id} incident={incident} />
      ))}
    </Carousel>
  );
}

function CarouselSkeleton() {
  return (
    <div className="flex gap-4 overflow-hidden" aria-hidden="true">
      {Array.from({ length: 3 }, (_, index) => (
        <div
          key={index}
          className="h-72 w-[85%] shrink-0 rounded-2xl border border-border bg-surface motion-safe:animate-pulse sm:w-[calc((100%-1rem)/2)] lg:w-[calc((100%-2rem)/3)]"
        />
      ))}
    </div>
  );
}

export function LatestIncidentsSection() {
  return (
    <section aria-labelledby="incidents-title" className="py-16 sm:py-20">
      <Container>
        <SectionHeading
          id="incidents-title"
          eyebrow="Mission accomplie"
          title="Derniers incidents traités avec succès"
          description="Chaque jour, des citoyens signalent et des héros interviennent. Voici les dernières interventions réussies."
          className="mb-8"
        />
        <Suspense fallback={<CarouselSkeleton />}>
          <LatestIncidentsCarousel />
        </Suspense>
      </Container>
    </section>
  );
}
