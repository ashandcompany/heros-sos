import type { Metadata } from "next";
import { Suspense } from "react";
import { AccountsModeration } from "@/components/admin/AccountsModeration";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";

export const metadata: Metadata = { title: "Comptes" };

export default function AdminAccountsPage({ searchParams }: PageProps<"/admin/comptes">) {
  return (
    <>
      <AdminPageHeader
        title="Comptes"
        description="Contrôle des créations de comptes et des changements d'adresse email des citoyens."
      />
      <Suspense fallback={<AdminListSkeleton />}>
        <AccountsModeration searchParams={searchParams} />
      </Suspense>
    </>
  );
}
