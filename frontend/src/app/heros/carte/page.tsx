import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import {
  HeroIncidentMapContent,
  HeroIncidentMapSkeleton,
} from "@/components/hero/HeroIncidentMapContent";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Carte" };

export default function HeroIncidentMapPage() {
  return (
    <>
      <AdminPageHeader
        title="Carte des incidents"
        description="Incidents non résolus, super-héros en intervention et zones touchées. Cliquez sur un repère pour ouvrir l'incident."
        action={
          <ButtonLink href={routes.heroIncidents} variant="secondary">
            Voir la liste
          </ButtonLink>
        }
      />
      <Suspense fallback={<HeroIncidentMapSkeleton />}>
        <HeroIncidentMapContent />
      </Suspense>
    </>
  );
}
