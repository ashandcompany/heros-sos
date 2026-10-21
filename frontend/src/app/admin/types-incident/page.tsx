import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { IncidentTypeForm } from "@/components/admin/IncidentTypeForm";
import { IncidentTypesList } from "@/components/admin/IncidentTypesList";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";

export const metadata: Metadata = { title: "Types d'incident" };

export default function AdminIncidentTypesPage({
  searchParams,
}: PageProps<"/admin/types-incident">) {
  return (
    <>
      <AdminPageHeader
        title="Types d'incident"
        description="Référentiel des types proposés à la déclaration d'un incident. Un type créé est aussitôt proposé dans les formulaires."
      />

      <section
        aria-labelledby="new-type-title"
        className="mb-12 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8"
      >
        <h2 id="new-type-title" className="mb-6 font-display text-xl font-semibold">
          Nouveau type
        </h2>
        <IncidentTypeForm />
      </section>

      <section aria-labelledby="types-title">
        <h2 id="types-title" className="mb-4 font-display text-xl font-semibold">
          Types existants
        </h2>
        <Suspense fallback={<AdminListSkeleton items={1} />}>
          <IncidentTypesList searchParams={searchParams} />
        </Suspense>
      </section>
    </>
  );
}
