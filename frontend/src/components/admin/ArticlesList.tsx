import Link from "next/link";
import { deleteArticleAction } from "@/app/actions/admin";
import { Badge } from "@/components/ui/Badge";
import { getAllArticles } from "@/lib/api";
import { formatDateTime } from "@/lib/format";
import { adminArticleRoute } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import { ConfirmDeleteButton } from "./ConfirmDeleteButton";
import { EmptyState } from "./ModerationFilter";

/** Articles du blog (brouillons compris). Lit la session : à rendre sous <Suspense>. */
export async function ArticlesList() {
  const token = await requireAdminSession();
  const articles = await getAllArticles(token);

  if (articles.length === 0) {
    return <EmptyState>Aucun article pour le moment. Rédigez le premier&nbsp;!</EmptyState>;
  }

  return (
    <ul className="space-y-3">
      {articles.map((article) => {
        const titleId = `article-${article.id}-titre`;
        return (
          <li key={article.id}>
            <article
              aria-labelledby={titleId}
              className="grid gap-4 rounded-2xl border border-border bg-surface p-5 sm:p-6 lg:grid-cols-[minmax(0,1fr)_auto] lg:items-center"
            >
              <div className="min-w-0">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge tone={article.status === "publie" ? "success" : "paper"}>
                    {article.status === "publie" ? "Publié" : "Brouillon"}
                  </Badge>
                  <span className="font-mono text-xs text-muted">/blog/{article.slug}</span>
                </div>
                <h2 id={titleId} className="mt-3 font-display text-lg font-semibold">
                  {article.title}
                </h2>
                <p className="mt-1 text-sm text-pretty text-muted">{article.excerpt}</p>
                <p className="mt-2 text-xs text-muted">
                  {article.authorName} · Modifié le{" "}
                  <time dateTime={article.updatedAt} className="font-mono">
                    {formatDateTime(article.updatedAt)}
                  </time>
                </p>
              </div>
              <div className="flex flex-wrap gap-2">
                <Link
                  href={adminArticleRoute(article.id)}
                  className="inline-flex min-h-11 items-center justify-center rounded-md border border-border px-4 text-sm font-semibold hover:border-muted hover:bg-surface-raised"
                >
                  Modifier<span className="sr-only"> : {article.title}</span>
                </Link>
                <ConfirmDeleteButton
                  action={deleteArticleAction}
                  id={article.id}
                  subject={article.title}
                  description="L'article sera définitivement retiré du blog. Cette action est irréversible."
                />
              </div>
            </article>
          </li>
        );
      })}
    </ul>
  );
}
