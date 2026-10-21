import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { Pagination, readPageNumber } from "@/components/admin/Pagination";
import { VillainCard } from "@/components/encyclopedia/VillainCard";
import { SuperHeroCard } from "@/components/home/SuperHeroCard";
import { Container } from "@/components/ui/Container";
import { SectionHeading } from "@/components/ui/SectionHeading";
import { getSuperHeroes, getVillainsPage } from "@/lib/api";
import { routes } from "@/lib/routes";

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

/** Multiple de 2 et de 3 : les pages pleines remplissent la grille à toutes les largeurs. */
const VILLAINS_PAGE_SIZE = 12;
/** Paramètre d'URL de la page des vilains (`?vilains=2`), distinct d'une future page des héros. */
const VILLAINS_PAGE_PARAM = "vilains";
const VILLAINS_ANCHOR = "super-vilains";

/** Lit l'URL : rendu sous <Suspense>. */
async function VillainsGrid({
  searchParams,
}: {
  searchParams: PageProps<"/encyclopedie">["searchParams"];
}) {
  const pageNumber = await readPageNumber(searchParams, VILLAINS_PAGE_PARAM);
  const page = await getVillainsPage({
    limit: VILLAINS_PAGE_SIZE,
    offset: (pageNumber - 1) * VILLAINS_PAGE_SIZE,
  });

  if (page.total === 0) {
    return <EmptyState>Aucun super-vilain répertorié pour l&apos;instant.</EmptyState>;
  }
  // Page au-delà de la dernière (lien ancien ou saisi à la main).
  if (page.items.length === 0) {
    return (
      <EmptyState>
        Cette page n&apos;existe pas.{" "}
        <Link
          href={`${routes.encyclopedia}#${VILLAINS_ANCHOR}`}
          className="text-accent hover:underline"
        >
          Revenir aux premiers super-vilains
        </Link>
      </EmptyState>
    );
  }
  return (
    <>
      <ul className="grid gap-4 sm:grid-cols-2 sm:gap-6 lg:grid-cols-3">
        {page.items.map((villain) => (
          <li key={villain.id}>
            <VillainCard villain={villain} />
          </li>
        ))}
      </ul>
      <Pagination
        page={page}
        basePath={routes.encyclopedia}
        param={VILLAINS_PAGE_PARAM}
        anchor={VILLAINS_ANCHOR}
      />
    </>
  );
}

function GridSkeleton() {
  return (
    <div aria-hidden="true" className={gridClasses}>
      {Array.from({ length: 4 }, (_, index) => (
        <div
          key={index}
          className="h-56 border-2 border-border bg-surface motion-safe:animate-pulse"
        />
      ))}
    </div>
  );
}

export default function EncyclopediaPage({ searchParams }: PageProps<"/encyclopedie">) {
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
          href={`#${VILLAINS_ANCHOR}`}
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
        id={VILLAINS_ANCHOR}
        aria-labelledby="villains-list-title"
        className="mt-16 scroll-mt-20"
      >
        <SectionHeading id="villains-list-title" title="Super-vilains" className="mb-6" />
        <Suspense fallback={<GridSkeleton />}>
          <VillainsGrid searchParams={searchParams} />
        </Suspense>
      </section>
    </Container>
  );
}
