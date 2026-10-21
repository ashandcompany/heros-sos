import type { Metadata } from "next";
import { Suspense } from "react";
import { heroDeclareIncidentAction } from "@/app/actions/incidents";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { DeclareIncidentForm } from "@/components/citizen/DeclareIncidentForm";
import { IncidentListItem } from "@/components/citizen/IncidentListItem";
import { ListSkeleton } from "@/components/citizen/IncidentsSections";
import { getMyIncidents, getVillains } from "@/lib/api";
import { requireHeroSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Signaler un incident" };

async function HeroDeclareIncident() {
  await requireHeroSession();
  const villains = await getVillains();
  return (
    <DeclareIncidentForm
      villains={villains.map(({ id, alias }) => ({ id, alias }))}
      action={heroDeclareIncidentAction}
    />
  );
}

async function MyDeclarations() {
  const { token } = await requireHeroSession();
  const incidents = await getMyIncidents(token);

  if (incidents.length === 0) {
    return <p className="text-sm text-muted">Vous n&apos;avez encore déclaré aucun incident.</p>;
  }

  return (
    <ul className="grid gap-3 md:grid-cols-2">
      {incidents.map((incident) => (
        <li key={incident.id}>
          <IncidentListItem incident={incident} detailed />
        </li>
      ))}
    </ul>
  );
}

export default function HeroDeclareIncidentPage() {
  return (
    <>
      <AdminPageHeader
        title="Signaler un incident"
        description="Témoin d'un méfait en dehors d'une intervention ? Déclarez-le comme n'importe quel citoyen : après vérification par la modération, les super-héros à proximité seront prévenus."
      />
      <section
        aria-label="Déclaration"
        className="max-w-3xl border-2 border-border bg-surface p-5 shadow-[3px_3px_0_0_var(--color-shadow)] sm:p-8"
      >
        <Suspense fallback={<div aria-hidden="true" className="h-[40rem]" />}>
          <HeroDeclareIncident />
        </Suspense>
      </section>

      <section aria-labelledby="mine-title" className="mt-12">
        <h2 id="mine-title" className="mb-4 font-display text-xl font-semibold">
          Mes déclarations
        </h2>
        <Suspense fallback={<ListSkeleton items={2} />}>
          <MyDeclarations />
        </Suspense>
      </section>
    </>
  );
}
