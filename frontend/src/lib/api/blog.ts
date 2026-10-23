import { cache } from "react";
import { articlesMock } from "@/mocks/articles";
import type { ArticleInput, BlogArticle, BlogArticleSummary, MutationResult } from "@/types";
import { apiFailureMessage } from "@/lib/validation/form-state";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { MAX_PAGE_SIZE, pageQuery } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Blog : articles publics (`/articles`, sans session) et back-office (`/admin/articles`, réservé
 * aux modérateurs). Le slug est calculé et le HTML assaini par l'API ; les listes ne contiennent
 * pas le contenu.
 */

/* ---------- Conversion depuis l'API ---------- */

function fromApiSummary(article: ApiSchema<"ArticleSummary">): BlogArticleSummary {
  return {
    id: article.id,
    title: article.title,
    slug: article.slug,
    excerpt: article.excerpt,
    status: article.status,
    authorName: article.authorName,
    createdAt: article.createdAt,
    updatedAt: article.updatedAt,
    publishedAt: article.publishedAt ?? null,
  };
}

function fromApiArticle(article: ApiSchema<"Article">): BlogArticle {
  return { ...fromApiSummary(article), content: article.content };
}

/** Toutes les pages d'une liste d'articles (100 par appel), dans l'ordre de l'API. */
async function fetchAllArticles(path: string, token?: string): Promise<BlogArticleSummary[]> {
  const articles: BlogArticleSummary[] = [];
  for (let offset = 0; ; offset += MAX_PAGE_SIZE) {
    const response = await apiFetch(`${path}${pageQuery({ limit: MAX_PAGE_SIZE, offset })}`, {
      sessionToken: token,
    });
    const page = (await response.json()) as ApiSchema<"ArticleSummaryPage">;
    articles.push(...page.items.map(fromApiSummary));
    if (page.items.length === 0 || articles.length >= page.total) return articles;
  }
}

/** Un article (`GET /articles/{slug}` ou `/admin/articles/{id}`), `null` sur un 404. */
async function fetchArticle(path: string, token?: string): Promise<BlogArticle | null> {
  try {
    const response = await apiFetch(path, { sessionToken: token });
    return fromApiArticle((await response.json()) as ApiSchema<"Article">);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}

function toArticleFailure(error: unknown, context: string): { ok: false; error: string } {
  if (error instanceof ApiError) {
    if (error.status === 404) return { ok: false, error: "Article introuvable." };
    if (error.code === "article_slug_already_used") {
      return {
        ok: false,
        error: "Impossible de générer une adresse unique pour cet article : modifiez le titre.",
      };
    }
    if (error.code === "validation_failed") {
      return { ok: false, error: "Certaines informations de l'article sont invalides." };
    }
  }
  return { ok: false, error: apiFailureMessage(toApiFailure(error, context)) };
}

/* ---------- Mocks ---------- */

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

/** Les listes de l'API ne contiennent pas le contenu : le mock fait de même. */
function toSummary({ content, ...summary }: BlogArticle): BlogArticleSummary {
  void content;
  return summary;
}

/* ---------- Back-office ---------- */

/**
 * Tous les articles (brouillons compris), les derniers modifiés d'abord, sans leur contenu.
 * API : `GET /admin/articles`, toutes les pages.
 */
export async function getAllArticles(token: string): Promise<BlogArticleSummary[]> {
  if (isApiEnabled) return fetchAllArticles("/admin/articles", token);

  await userFromToken(token, "admin");
  return mockResponse(
    [...articlesMock]
      .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt))
      .map(toSummary),
  );
}

/** Un article, brouillon compris. API : `GET /admin/articles/{id}` (404 → `null`). */
export async function getArticle(token: string, id: string): Promise<BlogArticle | null> {
  if (isApiEnabled) return fetchArticle(`/admin/articles/${encodeURIComponent(id)}`, token);

  await userFromToken(token, "admin");
  return mockResponse(articlesMock.find((article) => article.id === id) ?? null);
}

/**
 * Crée un article ; l'API calcule le slug et assainit `content` (balises de l'éditeur seulement).
 * API : `POST /admin/articles` avec l'`ArticleInput`.
 */
export async function createArticle(
  token: string,
  input: ArticleInput,
): Promise<MutationResult & { id?: string }> {
  if (isApiEnabled) {
    try {
      const response = await apiFetch("/admin/articles", {
        method: "POST",
        sessionToken: token,
        body: input satisfies ApiSchema<"ArticlePayload">,
      });
      return { ok: true, id: ((await response.json()) as ApiSchema<"Article">).id };
    } catch (error) {
      return toArticleFailure(error, "Création d'un article");
    }
  }

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

/**
 * Remplace titre, chapeau, contenu et statut. L'API fige le slug à la première publication.
 * API : `PUT /admin/articles/{id}` avec l'`ArticleInput`.
 */
export async function updateArticle(
  token: string,
  id: string,
  input: ArticleInput,
): Promise<MutationResult> {
  if (isApiEnabled) {
    try {
      await apiFetch(`/admin/articles/${encodeURIComponent(id)}`, {
        method: "PUT",
        sessionToken: token,
        body: input satisfies ApiSchema<"ArticlePayload">,
      });
      return { ok: true };
    } catch (error) {
      return toArticleFailure(error, "Modification d'un article");
    }
  }

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

/** API : `DELETE /admin/articles/{id}`. */
export async function deleteArticle(token: string, id: string): Promise<MutationResult> {
  if (isApiEnabled) {
    try {
      await apiFetch(`/admin/articles/${encodeURIComponent(id)}`, {
        method: "DELETE",
        sessionToken: token,
      });
      return { ok: true };
    } catch (error) {
      return toArticleFailure(error, "Suppression d'un article");
    }
  }

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
 * Articles publiés, du plus récent au plus ancien, sans leur contenu.
 * API : `GET /articles` (sans session), toutes les pages.
 */
export async function getPublishedArticles(): Promise<BlogArticleSummary[]> {
  if (isApiEnabled) return fetchAllArticles("/articles");

  return mockResponse(
    articlesMock
      .filter(isPublished)
      .sort((a, b) => Date.parse(b.publishedAt) - Date.parse(a.publishedAt))
      .map(toSummary),
  );
}

/**
 * Un article publié. API : `GET /articles/{slug}` (brouillon ou slug inconnu : 404 → `null`).
 * Mémorisé le temps d'un rendu : la page et ses métadonnées le lisent toutes les deux.
 */
export const getPublishedArticle = cache(async (slug: string): Promise<BlogArticle | null> => {
  if (isApiEnabled) return fetchArticle(`/articles/${encodeURIComponent(slug)}`);

  return mockResponse(
    articlesMock.find((article) => article.slug === slug && isPublished(article)) ?? null,
  );
});
