import type { Metadata } from "next";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { ArticlesList } from "@/components/admin/ArticlesList";
import { AdminListSkeleton } from "@/components/admin/ModerationFilter";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Blog" };

export default function AdminBlogPage() {
  return (
    <>
      <AdminPageHeader
        title="Blog"
        description="Rédigez, modifiez ou supprimez les articles du blog. Un brouillon n'est visible que dans le back-office."
        action={<ButtonLink href={routes.adminNewArticle}>Nouvel article</ButtonLink>}
      />
      <Suspense fallback={<AdminListSkeleton />}>
        <ArticlesList />
      </Suspense>
    </>
  );
}
