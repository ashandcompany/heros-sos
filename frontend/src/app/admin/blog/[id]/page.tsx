import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { deleteArticleAction } from "@/app/actions/admin";
import { ArticleForm } from "@/components/admin/ArticleForm";
import { ConfirmDeleteButton } from "@/components/admin/ConfirmDeleteButton";
import { getArticle } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";

export const metadata: Metadata = { title: "Modifier un article" };

/** Lit la session et les paramètres d'URL : rendu sous <Suspense>. */
async function ArticleEditor({ params }: { params: Promise<{ id: string }> }) {
  const token = await requireAdminSession();
  const { id } = await params;
  const article = await getArticle(token, id);
  if (!article) notFound();

  return (
    <>
      <div className="mb-8 flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="font-display text-3xl font-bold">Modifier l&apos;article</h1>
          <p className="mt-2 text-sm text-muted">
            Créé par {article.authorName} le{" "}
            <time dateTime={article.createdAt}>{formatDateTime(article.createdAt)}</time>
            {article.publishedAt && (
              <>
                {" "}
                · Publié le{" "}
                <time dateTime={article.publishedAt}>{formatDateTime(article.publishedAt)}</time>
              </>
            )}
          </p>
        </div>
        <ConfirmDeleteButton
          action={deleteArticleAction}
          id={article.id}
          subject={article.title}
          description="L'article sera définitivement retiré du blog. Cette action est irréversible."
        />
      </div>
      <div className="max-w-3xl border-2 border-border bg-surface shadow-[4px_4px_0_0_var(--color-shadow)] p-5 sm:p-8">
        <ArticleForm article={article} />
      </div>
    </>
  );
}

export default function AdminArticlePage({ params }: PageProps<"/admin/blog/[id]">) {
  return (
    <>
      <p className="mb-4 text-sm">
        <Link href={routes.adminBlog} className="text-accent hover:underline">
          ← Articles du blog
        </Link>
      </p>
      <Suspense
        fallback={
          <div
            aria-hidden="true"
            className="h-[40rem] border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        }
      >
        <ArticleEditor params={params} />
      </Suspense>
    </>
  );
}
