import type { Metadata } from "next";
import { Suspense } from "react";
import { ListSkeleton } from "@/components/citizen/IncidentsSections";
import { InterventionsList } from "@/components/citizen/InterventionsList";

export const metadata: Metadata = { title: "Noter un super-héros" };

export default function RateHeroPage() {
  return (
    <>
      <h1 className="font-display text-3xl font-bold">Noter un super-héros</h1>
      <p className="mt-2 mb-8 max-w-2xl text-muted">
        Retrouvez les interventions sur les incidents que vous avez déclarés et donnez votre avis :
        vos notes garantissent la qualité des interventions.
      </p>
      <Suspense fallback={<ListSkeleton items={2} />}>
        <InterventionsList />
      </Suspense>
    </>
  );
}
