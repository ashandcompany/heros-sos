import type { Metadata } from "next";
import { Suspense } from "react";
import {
  DeclareIncidentSection,
  ListSkeleton,
  MyIncidents,
  OngoingIncidents,
} from "@/components/citizen/IncidentsSections";

export const metadata: Metadata = { title: "Mes incidents" };

export default function MyIncidentsPage({ searchParams }: PageProps<"/profil/incidents">) {
  return (
    <>
      <h1 className="font-display text-3xl font-bold">Mes incidents</h1>
      <p className="mt-2 mb-8 max-w-2xl text-muted">
        Témoin ou victime d&apos;un super-vilain&nbsp;? Vérifiez d&apos;abord que l&apos;incident
        n&apos;est pas déjà signalé, puis déclarez-le : les super-héros à proximité seront prévenus.
      </p>

      <div className="grid gap-10 lg:grid-cols-[minmax(0,1fr)_minmax(0,1.3fr)] lg:items-start">
        <section aria-labelledby="ongoing-title">
          <h2 id="ongoing-title" className="mb-4 font-display text-xl font-semibold">
            Incidents en cours
          </h2>
          <Suspense fallback={<ListSkeleton />}>
            <OngoingIncidents searchParams={searchParams} />
          </Suspense>
        </section>

        <section
          aria-labelledby="declare-title"
          className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8"
        >
          <h2 id="declare-title" className="mb-6 font-display text-xl font-semibold">
            Déclarer un incident
          </h2>
          <Suspense fallback={<div aria-hidden="true" className="h-[40rem]" />}>
            <DeclareIncidentSection />
          </Suspense>
        </section>
      </div>

      <section aria-labelledby="mine-title" className="mt-12">
        <h2 id="mine-title" className="mb-4 font-display text-xl font-semibold">
          Mes déclarations
        </h2>
        <Suspense fallback={<ListSkeleton items={2} />}>
          <MyIncidents />
        </Suspense>
      </section>
    </>
  );
}
