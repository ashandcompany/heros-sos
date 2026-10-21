import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { ListSkeleton } from "@/components/citizen/IncidentsSections";
import { InterventionsList } from "@/components/citizen/InterventionsList";

export const metadata: Metadata = { title: "Noter un super-héros" };

/** Un héros qui a déclaré un incident note, comme un citoyen, les héros intervenus (pas lui-même). */
export default function HeroRatePage() {
  return (
    <>
      <AdminPageHeader
        title="Noter un super-héros"
        description="Vous avez signalé un incident ? Donnez votre avis sur les super-héros qui sont intervenus. Vous ne pouvez pas vous noter vous-même."
      />
      <Suspense fallback={<ListSkeleton items={2} />}>
        <InterventionsList role="super-heros" />
      </Suspense>
    </>
  );
}
