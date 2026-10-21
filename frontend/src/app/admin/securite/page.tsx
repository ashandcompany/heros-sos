import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { MfaPanel } from "@/components/admin/MfaPanel";
import { SessionsPanel } from "@/components/citizen/AccountDetails";

export const metadata: Metadata = { title: "Sécurité" };

/** Sécurité du compte du modérateur connecté (réservé au rôle admin : src/proxy.ts). */
export default function AdminSecurityPage() {
  return (
    <>
      <AdminPageHeader
        title="Sécurité"
        description="Protégez votre compte de modérateur : il donne accès à toutes les données de la plateforme."
      />
      <div className="max-w-2xl space-y-6">
        <Suspense
          fallback={
            <div
              aria-hidden="true"
              className="h-64 border-2 border-border bg-surface motion-safe:animate-pulse"
            />
          }
        >
          <MfaPanel />
        </Suspense>
        <SessionsPanel />
      </div>
    </>
  );
}
