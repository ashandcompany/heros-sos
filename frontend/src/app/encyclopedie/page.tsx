import type { Metadata } from "next";
import { Suspense } from "react";
import { EncyclopediaBrowser } from "@/components/encyclopedia/EncyclopediaBrowser";
import { Container } from "@/components/ui/Container";

export const metadata: Metadata = {
  title: "Encyclopédie",
  description: "Tous les super-héros inscrits sur Heros-SOS et les super-vilains connus.",
};

function EncyclopediaSkeleton() {
  return (
    <div aria-hidden="true">
      <div className="mt-8 h-12 w-80 max-w-full border-b-2 border-border" />
      <div className="mt-6 h-28 border-2 border-border bg-surface motion-safe:animate-pulse" />
      <div className="mt-10 grid grid-cols-2 gap-4 sm:gap-6 md:grid-cols-3 lg:grid-cols-4">
        {Array.from({ length: 8 }, (_, index) => (
          <div
            key={index}
            className="h-72 border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        ))}
      </div>
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

      <Suspense fallback={<EncyclopediaSkeleton />}>
        <EncyclopediaBrowser searchParams={searchParams} />
      </Suspense>
    </Container>
  );
}
