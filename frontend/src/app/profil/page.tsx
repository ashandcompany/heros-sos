import type { Metadata } from "next";
import { Suspense } from "react";
import { LogoutEverywhereForm } from "@/components/auth/LogoutEverywhereForm";
import { AccountDetails, AccountDetailsSkeleton, Panel } from "@/components/citizen/AccountDetails";
import { HeroRequestPanel } from "@/components/citizen/HeroRequestPanel";

export const metadata: Metadata = { title: "Mon compte" };

export default function MyAccountPage() {
  return (
    <>
      <h1 className="mb-8 font-display text-3xl font-bold">Mon compte</h1>
      <Suspense fallback={<AccountDetailsSkeleton />}>
        <AccountDetails />
      </Suspense>
      <div className="mt-6">
        <Suspense
          fallback={
            <div
              aria-hidden="true"
              className="h-64 border-2 border-border bg-surface motion-safe:animate-pulse"
            />
          }
        >
          <HeroRequestPanel />
        </Suspense>
      </div>
      <div className="mt-6">
        <Panel id="security-title" title="Sécurité">
          <LogoutEverywhereForm />
        </Panel>
      </div>
    </>
  );
}
