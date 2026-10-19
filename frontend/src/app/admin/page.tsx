import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { DashboardSummary, DashboardSummarySkeleton } from "@/components/admin/DashboardSummary";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Tableau de bord" };

export default function AdminDashboardPage() {
  return (
    <>
      <AdminPageHeader
        title="Tableau de bord"
        description="Vérifiez que chaque utilisateur respecte les règles : incidents signalés, comptes, demandes de super-héros, vidéothèques et commentaires."
        action={
          <ButtonLink href={routes.adminReportIncident} variant="secondary">
            Signaler un incident
          </ButtonLink>
        }
      />
      <Suspense fallback={<DashboardSummarySkeleton />}>
        <DashboardSummary />
      </Suspense>
    </>
  );
}
