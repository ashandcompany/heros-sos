import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { AdminPageHeader } from "@/components/admin/AdminPageHeader";
import { ArticleForm } from "@/components/admin/ArticleForm";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Nouvel article" };

async function NewArticle() {
  await requireAdminSession();
  return <ArticleForm />;
}

export default function AdminNewArticlePage() {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.adminBlog} className="text-accent hover:underline">
          ← Articles du blog
        </Link>
      </p>
      <AdminPageHeader title="Nouvel article" />
      <div className="max-w-3xl border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8">
        <Suspense fallback={<div aria-hidden="true" className="h-[40rem]" />}>
          <NewArticle />
        </Suspense>
      </div>
    </>
  );
}
