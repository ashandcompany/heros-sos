import type { Metadata } from "next";
import { Suspense } from "react";
import { VillainCard } from "@/components/encyclopedia/VillainCard";
import { SuperHeroCard } from "@/components/home/SuperHeroCard";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { getSuperHeroes, getVillains } from "@/lib/api";

export const metadata: Metadata = {
  title: "Encyclopédie",
  description: "Tous les super-héros inscrits sur Heros-SOS et les super-vilains connus.",
};

const gridClasses = "grid grid-cols-2 gap-4 sm:gap-6 md:grid-cols-3 lg:grid-cols-4";

async function HeroesGrid() {
  const heroes = await getSuperHeroes();
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

async function VillainsGrid() {
  const villains = await getVillains();
  return (
    <ul className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
      {villains.map((villain) => (
        <li key={villain.id}>
          <VillainCard villain={villain} />
        </li>
      ))}
    </ul>
  );
}

function GridSkeleton() {
  return (
    <div aria-hidden="true" className={gridClasses}>
      {Array.from({ length: 4 }, (_, index) => (
        <div
          key={index}
          className="h-56 rounded-2xl border border-border bg-surface motion-safe:animate-pulse"
        />
      ))}
    </div>
  );
}

export default function EncyclopediaPage() {
  return (
    <Container className="py-10 sm:py-14">
      <h1 className="font-display text-3xl font-bold sm:text-4xl">Encyclopédie</h1>
      <p className="mt-2 max-w-2xl text-muted">
        Les super-héros qui veillent sur vous et les super-vilains connus. Ces fiches sont enrichies
        au fil des déclarations des citoyens et par l&apos;équipe de modération.
      </p>

      <nav aria-label="Sections de l'encyclopédie" className="mt-6 flex flex-wrap gap-2">
        <a
          href="#super-heros"
          className="rounded-full border border-border px-4 py-2 text-sm font-medium hover:bg-surface-raised"
        >
          Super-héros
        </a>
        <a
          href="#super-vilains"
          className="rounded-full border border-border px-4 py-2 text-sm font-medium hover:bg-surface-raised"
        >
          Super-vilains
        </a>
      </nav>

      <section id="super-heros" aria-labelledby="heroes-list-title" className="mt-12 scroll-mt-20">
        <SectionHeading id="heroes-list-title" title="Super-héros" className="mb-6" />
        <Suspense fallback={<GridSkeleton />}>
          <HeroesGrid />
        </Suspense>
      </section>

      <section
        id="super-vilains"
        aria-labelledby="villains-list-title"
        className="mt-16 scroll-mt-20"
      >
        <SectionHeading id="villains-list-title" title="Super-vilains" className="mb-6" />
        <Suspense fallback={<GridSkeleton />}>
          <VillainsGrid />
        </Suspense>
      </section>
    </Container>
  );
}
