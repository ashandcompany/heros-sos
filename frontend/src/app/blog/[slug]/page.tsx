import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { RichContent } from "@/components/ui/RichContent";
import { getPublishedArticle } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { blogArticleRoute, routes } from "@/lib/routes";
import { getSiteUrl } from "@/lib/site";

export async function generateMetadata({ params }: PageProps<"/blog/[slug]">): Promise<Metadata> {
  const article = await getPublishedArticle((await params).slug);
  if (!article?.publishedAt) return { title: "Article introuvable" };

  // Rendu à la requête (paramètre d'URL) : SITE_URL n'est pas figée au build.
  return {
    metadataBase: new URL(getSiteUrl()),
    title: article.title,
    description: article.excerpt,
    alternates: { canonical: blogArticleRoute(article.slug) },
    openGraph: {
      type: "article",
      title: article.title,
      description: article.excerpt,
      publishedTime: article.publishedAt,
      authors: [article.authorName],
    },
  };
}

/** Lit les paramètres d'URL : rendu sous <Suspense>. */
async function Article({ params }: { params: Promise<{ slug: string }> }) {
  const article = await getPublishedArticle((await params).slug);
  if (!article?.publishedAt) notFound();

  return (
    <article aria-labelledby="article-title">
      <header className="max-w-3xl">
        <h1 id="article-title" className="font-display text-3xl font-bold text-balance sm:text-4xl">
          {article.title}
        </h1>
        <p className="mt-3 text-lg text-pretty text-muted">{article.excerpt}</p>
        <p className="mt-4 text-sm text-muted">
          Par <span className="font-medium text-foreground">{article.authorName}</span> · Publié le{" "}
          <time dateTime={article.publishedAt}>{formatDate(article.publishedAt)}</time>
        </p>
        <div aria-hidden="true" className="rule-thick mt-6 w-24" />
      </header>

      <RichContent html={article.content} className="text-lg" />
    </article>
  );
}

export default function BlogArticlePage({ params }: PageProps<"/blog/[slug]">) {
  return (
    <Container className="py-10 sm:py-14">
      <p className="mb-6 text-sm">
        <Link href={routes.blog} className="text-accent hover:underline">
          ← Tous les articles
        </Link>
      </p>
      <Suspense
        fallback={
          <div
            aria-hidden="true"
            className="h-96 max-w-3xl border-2 border-border bg-surface motion-safe:animate-pulse"
          />
        }
      >
        <Article params={params} />
      </Suspense>
    </Container>
  );
}
