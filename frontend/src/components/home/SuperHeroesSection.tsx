import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { getFeaturedSuperHeroes } from "@/lib/api";
import { SuperHeroCard } from "./SuperHeroCard";

const gridClasses = "grid grid-cols-2 gap-4 sm:gap-6 md:grid-cols-3 lg:grid-cols-4";

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
        <div
          key={index}
          className="h-64 rounded-2xl border border-border bg-surface motion-safe:animate-pulse"
        />
      ))}
    </div>
  );
}

export function SuperHeroesSection() {
  return (
    <section aria-labelledby="heroes-title" className="py-16 sm:py-20">
      <Container>
        <SectionHeading
          id="heroes-title"
          eyebrow="Ils veillent sur vous"
          title="Des super-héros vérifiés et notés par les citoyens"
          description="Chaque héros inscrit est évalué après chaque intervention. Voici ceux qui inspirent le plus confiance."
          className="mb-10"
        />
        <Suspense fallback={<SuperHeroesGridSkeleton />}>
          <SuperHeroesGrid />
        </Suspense>
      </Container>
    </section>
  );
}
