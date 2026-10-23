import { connection } from "next/server";
import { getPublishedArticle } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { defaultOgImage, ogContentType, ogImage, ogSize } from "@/lib/og";

export const alt = "Article du blog Heros-SOS";
export const size = ogSize;
export const contentType = ogContentType;

export default async function OpengraphImage({ params }: { params: Promise<{ slug: string }> }) {
  // Article lu à la requête (API ou mocks), jamais au build.
  await connection();
  const article = await getPublishedArticle((await params).slug);
  if (!article?.publishedAt) return defaultOgImage();

  return ogImage({
    kicker: "Blog",
    title: article.title,
    subtitle: `${article.authorName} · ${formatDate(article.publishedAt)}`,
  });
}
