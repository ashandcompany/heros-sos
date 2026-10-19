import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { HeroApplications } from "@/components/admin/HeroApplications";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import {
  SuperHeroesDirectory,
  SuperHeroesDirectorySkeleton,
} from "@/components/admin/SuperHeroesDirectory";

export const metadata: Metadata = { title: "Super-héros" };

/** Back-office, réservé aux administrateurs (src/proxy.ts + vérification dans les composants). */
export default function AdminSuperHeroesPage() {
  return (
    <>
      <AdminPageHeader
        title="Super-héros"
        description="Validez ou refusez les demandes des nouveaux super-héros et consultez le référentiel. Ces informations sont confidentielles : ne les partagez pas en dehors de l'équipe."
      />

      <section aria-labelledby="applications-title" className="mb-12">
        <h2 id="applications-title" className="mb-4 font-display text-xl font-semibold">
          Demandes en attente
        </h2>
        <Suspense fallback={<AdminListSkeleton items={2} />}>
          <HeroApplications />
        </Suspense>
      </section>

      <section aria-labelledby="directory-title">
        <h2 id="directory-title" className="mb-4 font-display text-xl font-semibold">
          Super-héros inscrits
        </h2>
        <Suspense fallback={<SuperHeroesDirectorySkeleton />}>
          <SuperHeroesDirectory />
        </Suspense>
      </section>
    </>
  );
}
