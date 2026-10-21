import type { Metadata } from "next";
import { Suspense } from "react";
import { LogoutEverywhereForm } from "@/components/auth/LogoutEverywhereForm";
import { AccountDetailsSkeleton, Panel } from "@/components/citizen/AccountDetails";
import { HeroProfileDetails } from "@/components/hero/HeroProfileDetails";

export const metadata: Metadata = { title: "Mon profil" };

export default function HeroProfilePage() {
  return (
    <>
      <h1 className="mb-8 font-display text-3xl font-bold">Mon profil</h1>
      <Suspense fallback={<AccountDetailsSkeleton />}>
        <HeroProfileDetails />
      </Suspense>
      <div className="mt-6">
        <Panel id="security-title" title="Sécurité">
          <LogoutEverywhereForm />
        </Panel>
      </div>
    </>
  );
}
