import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { IncidentsModeration } from "@/components/admin/IncidentsModeration";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { incidentFeatures } from "@/lib/api";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Incidents" };

export default function AdminIncidentsPage({ searchParams }: PageProps<"/admin/incidents">) {
  return (
    <>
      <AdminPageHeader
        title="Incidents signalés"
        description={
          incidentFeatures.moderation
            ? "Analysez chaque déclaration : validez-la si elle est conforme et réelle, corrigez les informations inexactes, rejetez les faux signalements (ils ne sont plus diffusés)."
            : "Incidents signalés, en cours ou en échec, et les équipes qui interviennent. Arbitrez les résultats contestés depuis le détail d'un incident."
        }
        action={
          <ButtonLink href={routes.adminReportIncident} variant="secondary">
            Signaler un incident
          </ButtonLink>
        }
      />
      <Suspense fallback={<AdminListSkeleton />}>
        <IncidentsModeration searchParams={searchParams} />
      </Suspense>
    </>
  );
}
