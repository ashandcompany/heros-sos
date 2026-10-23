import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { NewsletterForm } from "@/components/admin/NewsletterForm";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Nouvelle newsletter" };

async function NewNewsletter() {
  await requireAdminSession();
  return <NewsletterForm />;
}

export default function AdminNewNewsletterPage() {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.adminNewsletters} className="text-accent hover:underline">
          ← Newsletters
        </Link>
      </p>
      <AdminPageHeader
        title="Nouvelle newsletter"
        description="Le brouillon n'est envoyé à personne : vous pourrez le relire et le modifier avant l'envoi."
      />
      <div className="max-w-3xl border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8">
        <Suspense fallback={<div aria-hidden="true" className="h-[32rem]" />}>
          <NewNewsletter />
        </Suspense>
      </div>
    </>
  );
}
