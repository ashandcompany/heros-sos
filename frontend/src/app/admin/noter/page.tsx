import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { ListSkeleton } from "@/components/citizen/IncidentsSections";
import { InterventionsList } from "@/components/citizen/InterventionsList";

export const metadata: Metadata = { title: "Noter un super-héros" };

/** Un modérateur qui a signalé un incident note et commente les héros intervenus, comme un citoyen. */
export default function AdminRatePage() {
  return (
    <>
      <AdminPageHeader
        title="Noter un super-héros"
        description="Donnez votre avis sur les super-héros intervenus sur les incidents que vous avez signalés. Un avis commenté est validé par un autre modérateur."
      />
      <Suspense fallback={<ListSkeleton items={2} />}>
        <InterventionsList role="admin" />
      </Suspense>
    </>
  );
}
