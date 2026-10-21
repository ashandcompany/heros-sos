import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { Suspense } from "react";
import { Container } from "@/components/ui/Container";
import { getPublishedArticle } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { routes } from "@/lib/routes";
import { sanitizeArticleHtml } from "@/lib/sanitize";

export async function generateMetadata({ params }: PageProps<"/blog/[slug]">): Promise<Metadata> {
  const article = await getPublishedArticle((await params).slug);
  return article
    ? { title: article.title, description: article.excerpt }
    : { title: "Article introuvable" };
}

/** Mêmes styles que dans l'éditeur du back-office, à l'échelle d'une page de lecture. */
const contentClasses = [
  "max-w-prose text-lg leading-relaxed",
  "[&_a]:text-accent [&_a]:underline [&_blockquote]:my-6 [&_blockquote]:border-l-2 [&_blockquote]:border-accent [&_blockquote]:pl-4 [&_blockquote]:italic",
  "[&_h2]:mt-10 [&_h2]:font-display [&_h2]:text-2xl [&_h2]:font-bold [&_h3]:mt-8 [&_h3]:font-display [&_h3]:text-xl [&_h3]:font-semibold",
  "[&_hr]:my-8 [&_hr]:border-border [&_li]:my-1 [&_ol]:list-decimal [&_ol]:pl-6 [&_p]:my-4 [&_ul]:list-disc [&_ul]:pl-6",
].join(" ");

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

      <div
        className={contentClasses}
        // Contenu assaini (liste blanche de balises), voir src/lib/sanitize.ts.
        dangerouslySetInnerHTML={{ __html: sanitizeArticleHtml(article.content) }}
      />
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
