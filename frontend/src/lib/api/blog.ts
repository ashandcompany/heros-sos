import { articlesMock } from "@/mocks/articles";
import type { ArticleInput, BlogArticle, MutationResult } from "@/types";
import { mockResponse } from "./config";
import { userFromToken } from "./session-user";

/** « Héros & vilains : l'été ! » → « heros-vilains-l-ete » */
function slugify(title: string): string {
  return title
    .normalize("NFD")
    .replace(/\p{Diacritic}/gu, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}

/** Slug unique : suffixe numérique si un autre article l'utilise déjà. */
function uniqueSlug(title: string, ignoreId?: string): string {
  const base = slugify(title) || "article";
  const taken = (slug: string) => articlesMock.some((a) => a.slug === slug && a.id !== ignoreId);
  let slug = base;
  for (let suffix = 2; taken(slug); suffix++) slug = `${base}-${suffix}`;
  return slug;
}

/**
 * Tous les articles (brouillons compris), les derniers modifiés d'abord (back-office).
 * Future implémentation : `GET ${API_URL}/admin/articles?sort=-updatedAt`
 */
export async function getAllArticles(token: string): Promise<BlogArticle[]> {
  await userFromToken(token, "admin");
  return mockResponse(
    [...articlesMock].sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt)),
  );
}

/** Future implémentation : `GET ${API_URL}/admin/articles/${id}` (404 → `null`). */
export async function getArticle(token: string, id: string): Promise<BlogArticle | null> {
  await userFromToken(token, "admin");
  return mockResponse(articlesMock.find((article) => article.id === id) ?? null);
}

/**
 * L'API devra assainir `content` (liste blanche de balises) avant de l'enregistrer.
 * Future implémentation : `POST ${API_URL}/admin/articles` avec l'`ArticleInput`.
 */
export async function createArticle(
  token: string,
  input: ArticleInput,
): Promise<MutationResult & { id?: string }> {
  const user = await userFromToken(token, "admin");
  const now = new Date().toISOString();
  const id = `article-${crypto.randomUUID()}`;

  articlesMock.push({
    id,
    ...input,
    slug: uniqueSlug(input.title),
    authorName: user.displayName,
    createdAt: now,
    updatedAt: now,
    publishedAt: input.status === "publie" ? now : null,
  });

  return mockResponse({ ok: true, id });
}

/** Future implémentation : `PUT ${API_URL}/admin/articles/${id}` avec l'`ArticleInput`. */
export async function updateArticle(
  token: string,
  id: string,
  input: ArticleInput,
): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const article = articlesMock.find((existing) => existing.id === id);
  if (!article) return mockResponse({ ok: false, error: "Article introuvable." });

  const now = new Date().toISOString();
  Object.assign(article, input, {
    slug: uniqueSlug(input.title, id),
    updatedAt: now,
    publishedAt: input.status === "publie" ? (article.publishedAt ?? now) : article.publishedAt,
  });

  return mockResponse({ ok: true });
}

/** Future implémentation : `DELETE ${API_URL}/admin/articles/${id}` */
export async function deleteArticle(token: string, id: string): Promise<MutationResult> {
  await userFromToken(token, "admin");
  const index = articlesMock.findIndex((existing) => existing.id === id);
  if (index === -1) return mockResponse({ ok: false, error: "Article introuvable." });

  articlesMock.splice(index, 1);
  return mockResponse({ ok: true });
}

/* ---------- Blog public ---------- */

const isPublished = (article: BlogArticle): article is BlogArticle & { publishedAt: string } =>
  article.status === "publie" && !!article.publishedAt;

/**
 * Articles publiés, du plus récent au plus ancien.
 * Future implémentation : `GET ${API_URL}/articles?sort=-publishedAt`
 */
export async function getPublishedArticles(): Promise<BlogArticle[]> {
  return mockResponse(
    articlesMock
      .filter(isPublished)
      .sort((a, b) => Date.parse(b.publishedAt) - Date.parse(a.publishedAt)),
  );
}

/** Future implémentation : `GET ${API_URL}/articles/${slug}` (404 → `null`, brouillons compris). */
export async function getPublishedArticle(slug: string): Promise<BlogArticle | null> {
  return mockResponse(
    articlesMock.find((article) => article.slug === slug && isPublished(article)) ?? null,
  );
}
