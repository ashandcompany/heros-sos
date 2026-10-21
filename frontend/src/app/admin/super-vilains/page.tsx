import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { VillainForm } from "@/components/admin/VillainForm";
import { VillainsList } from "@/components/admin/VillainsList";
import { getIncidentTypes } from "@/lib/api";
import { requireAdminSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Super-vilains" };

/** Lit la session : rendu sous <Suspense>. */
async function NewVillainForm() {
  const token = await requireAdminSession();
  const types = await getIncidentTypes(token);
  return <VillainForm types={types} />;
}

export default function AdminVillainsPage({ searchParams }: PageProps<"/admin/super-vilains">) {
  return (
    <>
      <AdminPageHeader
        title="Super-vilains"
        description="Fiches de l'encyclopédie, proposées dans les formulaires de déclaration d'incident. Une fiche créée ou modifiée est aussitôt publiée."
      />

      <section
        aria-labelledby="new-villain-title"
        className="mb-12 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8"
      >
        <h2 id="new-villain-title" className="mb-6 font-display text-xl font-semibold">
          Nouvelle fiche
        </h2>
        <Suspense fallback={<AdminListSkeleton items={1} />}>
          <NewVillainForm />
        </Suspense>
      </section>

      <section aria-labelledby="villains-title">
        <h2 id="villains-title" className="mb-4 font-display text-xl font-semibold">
          Super-vilains répertoriés
        </h2>
        <Suspense fallback={<AdminListSkeleton items={1} />}>
          <VillainsList searchParams={searchParams} />
        </Suspense>
      </section>
    </>
  );
}
