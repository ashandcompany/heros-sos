import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { HeroReputationContent } from "@/components/hero/HeroReputationContent";

export const metadata: Metadata = { title: "Ma réputation" };

export default function HeroReputationPage() {
  return (
    <>
      <AdminPageHeader
        title="Ma réputation"
        description="Les notes et avis laissés par les citoyens après vos interventions. Les commentaires sont publiés après vérification par la modération."
      />
      <Suspense fallback={<AdminListSkeleton items={2} />}>
        <HeroReputationContent />
      </Suspense>
    </>
  );
}
