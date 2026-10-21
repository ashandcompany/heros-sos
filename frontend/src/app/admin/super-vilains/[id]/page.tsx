import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { VillainFilms } from "@/components/admin/VillainFilms";
import { VillainForm } from "@/components/admin/VillainForm";
import { getIncidentTypes, getVillain } from "@/lib/api";
import { formatInteger } from "@/lib/format";
import { routes, villainSheetRoute } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Fiche d'un super-vilain" };

/** Lit la session et les paramètres d'URL (`?q=` : recherche de films) : rendu sous <Suspense>. */
async function VillainEditor({ params, searchParams }: PageProps<"/admin/super-vilains/[id]">) {
  const token = await requireAdminSession();
  const [{ id }, { q }] = await Promise.all([params, searchParams]);
  const query = (Array.isArray(q) ? q[0] : q)?.trim() ?? "";
  const [villain, types] = await Promise.all([getVillain(id), getIncidentTypes(token)]);
  if (!villain) notFound();

  return (
    <>
      <div className="mb-8 flex flex-wrap items-start justify-between gap-4">
        <div>
          <h1 className="font-display text-3xl font-bold">{villain.alias}</h1>
          <p className="mt-2 text-sm text-muted">
            <span className="font-mono">{formatInteger(villain.reportedIncidentsCount)}</span>{" "}
            incident{villain.reportedIncidentsCount > 1 ? "s" : ""} signalé
            {villain.reportedIncidentsCount > 1 ? "s" : ""}
          </p>
        </div>
        <Link href={villainSheetRoute(villain.id)} className="text-sm text-accent hover:underline">
          Voir la fiche publique
        </Link>
      </div>

      <section
        aria-labelledby="edit-title"
        className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8"
      >
        <h2 id="edit-title" className="mb-6 font-display text-xl font-semibold">
          Informations du super-vilain
        </h2>
        <VillainForm villain={villain} types={types} />
      </section>

      <VillainFilms token={token} villain={villain} query={query} />
    </>
  );
}

export default function AdminVillainPage({
  params,
  searchParams,
}: PageProps<"/admin/super-vilains/[id]">) {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.adminVillains} className="text-accent hover:underline">
          ← Super-vilains
        </Link>
      </p>
      <Suspense
        fallback={
          <div
            aria-hidden="true"
            className="h-[40rem] border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        }
      >
        <VillainEditor params={params} searchParams={searchParams} />
      </Suspense>
    </>
  );
}
