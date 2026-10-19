import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { WallPoster } from "@/components/ui/WallPoster";
import { getFeaturedSuperHeroes } from "@/lib/api";
import { SuperHeroCard } from "./SuperHeroCard";

const gridClasses = "grid grid-cols-2 gap-4 sm:gap-8 md:grid-cols-3 lg:grid-cols-4";

async function SuperHeroesGrid() {
  const heroes = await getFeaturedSuperHeroes();

  return (
    <ul className={gridClasses}>
      {heroes.map((hero) => (
        <li key={hero.id}>
          <SuperHeroCard hero={hero} />
        </li>
      ))}
    </ul>
  );
}

function SuperHeroesGridSkeleton() {
  return (
    <div className={gridClasses} aria-hidden="true">
      {Array.from({ length: 8 }, (_, index) => (
        <div key={index} className="panel h-72 opacity-40 motion-safe:animate-pulse" />
      ))}
    </div>
  );
}

export function SuperHeroesSection() {
  return (
    <section aria-labelledby="heroes-title" className="py-14 sm:py-20">
      <Container>
        <div className="mb-10 flex items-start gap-10">
          <WallPoster
            src="/affiche3.jpg"
            width={736}
            height={981}
            caption="Note de service — cafétéria du personnel"
            className="-rotate-2"
          />
          <SectionHeading
            id="heroes-title"
            section="03"
            eyebrow="Registre du personnel"
            title="Agents vérifiés et notés par les citoyens"
            description="Chaque agent inscrit au registre est évalué après chaque intervention. Voici le personnel ayant reçu les meilleures appréciations."
            className="self-center"
          />
        </div>
        <Suspense fallback={<SuperHeroesGridSkeleton />}>
          <SuperHeroesGrid />
        </Suspense>
      </Container>
    </section>
  );
}
