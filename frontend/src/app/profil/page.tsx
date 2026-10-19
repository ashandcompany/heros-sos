import type { Metadata } from "next";
import { Suspense } from "react";
import { AccountDetails, AccountDetailsSkeleton } from "@/components/citizen/AccountDetails";

export const metadata: Metadata = { title: "Mon compte" };

export default function MyAccountPage() {
  return (
    <>
      <h1 className="mb-8 font-display text-3xl font-bold">Mon compte</h1>
      <Suspense fallback={<AccountDetailsSkeleton />}>
        <AccountDetails />
      </Suspense>
    </>
  );
}
