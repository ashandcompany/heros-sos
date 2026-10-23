import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { NewslettersList } from "@/components/admin/NewslettersList";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Newsletter" };

export default function AdminNewslettersPage({ searchParams }: PageProps<"/admin/newsletter">) {
  return (
    <>
      <AdminPageHeader
        title="Newsletter"
        description="Rédigez un brouillon, relisez-le, puis envoyez-le aux abonnés confirmés. Une newsletter envoyée reste consultable ici, en lecture seule."
        action={<ButtonLink href={routes.adminNewNewsletter}>Nouvelle newsletter</ButtonLink>}
      />
      <Suspense fallback={<AdminListSkeleton />}>
        <NewslettersList searchParams={searchParams} />
      </Suspense>
    </>
  );
}
