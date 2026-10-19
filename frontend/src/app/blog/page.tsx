import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { EmptyState } from "@/components/admin/ModerationFilter";
import { Container } from "@/components/ui/Container";
import { getPublishedArticles } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { blogArticleRoute } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Blog",
  description:
    "Conseils de sécurité, portraits de super-héros et bilans des interventions de Heros-SOS.",
};

async function ArticlesList() {
  const articles = await getPublishedArticles();

  if (articles.length === 0) {
    return <EmptyState>Aucun article publié pour l&apos;instant. Revenez bientôt !</EmptyState>;
  }

  return (
    <ul className="grid gap-4 sm:gap-6 md:grid-cols-2">
      {articles.map((article) => (
        <li key={article.id}>
          <article className="relative flex h-full flex-col border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-primary hover:border-muted sm:p-6">
            <p className="font-mono text-xs tracking-widest text-muted uppercase">
              <time dateTime={article.publishedAt ?? undefined}>
                {article.publishedAt && formatDate(article.publishedAt)}
              </time>
            </p>
            <h2 className="mt-2 font-display text-xl font-semibold text-balance">
              {/* Lien étiré : toute la carte est cliquable, le titre reste le nom du lien. */}
              <Link
                href={blogArticleRoute(article.slug)}
                className="after:absolute after:inset-0 after:rounded-2xl focus-visible:outline-none"
              >
                {article.title}
              </Link>
            </h2>
            <p className="mt-2 text-pretty text-muted">{article.excerpt}</p>
            <p aria-hidden="true" className="mt-auto pt-4 text-sm font-semibold text-accent">
              Lire l&apos;article&nbsp;→
            </p>
          </article>
        </li>
      ))}
    </ul>
  );
}

export default function BlogPage() {
  return (
    <Container className="py-10 sm:py-14">
      <h1 className="font-display text-3xl font-bold sm:text-4xl">Blog</h1>
      <p className="mt-2 mb-8 max-w-2xl text-muted">
        Conseils de sécurité, portraits de super-héros et bilans des interventions, par
        l&apos;équipe Heros-SOS.
      </p>
      <Suspense
        fallback={
          <div aria-hidden="true" className="grid gap-4 sm:gap-6 md:grid-cols-2">
            {Array.from({ length: 2 }, (_, index) => (
              <div
                key={index}
                className="h-48 border-2 border-border bg-surface motion-safe:animate-pulse"
              />
            ))}
          </div>
        }
      >
        <ArticlesList />
      </Suspense>
    </Container>
  );
}
