import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { ModeratorProfileDetails } from "@/components/admin/ModeratorProfileDetails";
import { AccountDetailsSkeleton } from "@/components/citizen/AccountDetails";

export const metadata: Metadata = { title: "Mon profil" };

/** Profil du modérateur connecté (réservé au rôle admin : src/proxy.ts + le composant). */
export default function AdminProfilePage() {
  return (
    <>
      <AdminPageHeader title="Mon profil" />
      <Suspense fallback={<AccountDetailsSkeleton />}>
        <ModeratorProfileDetails />
      </Suspense>
    </>
  );
}
