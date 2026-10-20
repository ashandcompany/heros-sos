import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { IncidentDetail } from "@/components/citizen/IncidentDetail";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Détail d'un incident" };

export default function CitizenIncidentPage({ params }: PageProps<"/profil/incidents/[id]">) {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.profileIncidents} className="text-accent hover:underline">
          ← Mes incidents
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
        <IncidentDetail params={params} />
      </Suspense>
    </>
  );
}
