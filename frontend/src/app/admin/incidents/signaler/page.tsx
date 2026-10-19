import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { adminDeclareIncidentAction } from "@/app/actions/incidents";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { DeclareIncidentForm } from "@/components/citizen/DeclareIncidentForm";
import { getVillains } from "@/lib/api";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Signaler un incident" };

async function AdminDeclareIncident() {
  await requireAdminSession();
  const villains = await getVillains();
  return (
    <DeclareIncidentForm
      villains={villains.map(({ id, alias }) => ({ id, alias }))}
      action={adminDeclareIncidentAction}
    />
  );
}

export default function AdminReportIncidentPage() {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.adminIncidents} className="text-accent hover:underline">
          ← Incidents signalés
        </Link>
      </p>
      <AdminPageHeader
        title="Signaler un incident"
        description="Même formulaire que celui des citoyens. Un incident signalé par la modération est validé d'office et diffusé immédiatement aux super-héros à proximité."
      />
      <section
        aria-label="Déclaration"
        className="max-w-3xl border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 sm:p-8"
      >
        <Suspense fallback={<div aria-hidden="true" className="h-[40rem]" />}>
          <AdminDeclareIncident />
        </Suspense>
      </section>
    </>
  );
}
