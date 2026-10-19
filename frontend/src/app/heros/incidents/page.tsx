import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { HeroIncidentsList } from "@/components/hero/HeroIncidentsList";

export const metadata: Metadata = { title: "Incidents" };

export default function HeroIncidentsPage({ searchParams }: PageProps<"/heros/incidents">) {
  return (
    <>
      <AdminPageHeader
        title="Incidents signalés"
        description="Consultez les signalements des citoyens, des plus urgents aux moins urgents. Un incident déjà pris en charge par un autre super-héros est indiqué comme tel."
      />
      <Suspense fallback={<AdminListSkeleton />}>
        <HeroIncidentsList searchParams={searchParams} />
      </Suspense>
    </>
  );
}
