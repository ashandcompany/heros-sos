import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { HeroIncidentDetail } from "@/components/hero/HeroIncidentDetail";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Détail d'un incident" };

export default function HeroIncidentPage({ params }: PageProps<"/heros/incidents/[id]">) {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.heroIncidents} className="text-accent hover:underline">
          ← Incidents signalés
        </Link>
      </p>
      <Suspense
        fallback={
          <div
            aria-hidden="true"
            className="h-[32rem] border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        }
      >
        <HeroIncidentDetail params={params} />
      </Suspense>
    </>
  );
}
