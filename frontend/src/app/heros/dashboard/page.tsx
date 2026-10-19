import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import {
  HeroDashboardContent,
  HeroDashboardSkeleton,
} from "@/components/hero/HeroDashboardContent";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Tableau de bord" };

export default function HeroDashboardPage() {
  return (
    <>
      <AdminPageHeader
        title="Tableau de bord"
        description="Vos interventions en cours, vos statistiques et celles de la plateforme, et les derniers signalements."
        action={
          <ButtonLink href={routes.heroIncidents} variant="secondary">
            Voir les incidents
          </ButtonLink>
        }
      />
      <Suspense fallback={<HeroDashboardSkeleton />}>
        <HeroDashboardContent />
      </Suspense>
    </>
  );
}
