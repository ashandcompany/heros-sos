import { describe, expect, it } from "vitest";
import type { ApiSchema } from "./http";
import { errorResponse, jsonResponse, mockFetch, withApi } from "./test-utils";

const summary: ApiSchema<"ArticleSummary"> = {
  id: "article-1",
  title: "Se protéger d'un rayon gelant",
  slug: "se-proteger-d-un-rayon-gelant",
  excerpt: "Les bons réflexes.",
  status: "publie",
  authorName: "Équipe Heros-SOS",
  createdAt: "2026-10-01T08:00:00Z",
  updatedAt: "2026-10-02T08:00:00Z",
  publishedAt: "2026-10-02T08:00:00Z",
};

describe("blog (API)", () => {
  it("lit toutes les pages de GET /articles", async () => {
    const second = { ...summary, id: "article-2", slug: "deuxieme" };
    const fetchMock = mockFetch((path) =>
      jsonResponse(
        path.includes("offset=0")
          ? { items: [summary], total: 2, limit: 100, offset: 0 }
          : { items: [second], total: 2, limit: 100, offset: 100 },
      ),
    );
    const { getPublishedArticles } = await withApi(() => import("./blog"));

    const articles = await getPublishedArticles();

    expect(articles.map(({ id }) => id)).toEqual(["article-1", "article-2"]);
    expect(articles[0]).toEqual(summary);
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });

  it("convertit un article avec son contenu", async () => {
    const article: ApiSchema<"Article"> = { ...summary, content: "<p>Restez à couvert.</p>" };
    const fetchMock = mockFetch(() => jsonResponse(article));
    const { getPublishedArticle } = await withApi(() => import("./blog"));

    expect(await getPublishedArticle("se proteger")).toEqual(article);
    expect(fetchMock.mock.calls[0][0]).toContain("/articles/se%20proteger");
  });

  it("renvoie null pour un article inconnu (404)", async () => {
    mockFetch(() => errorResponse(404, "article_not_found"));
    const { getPublishedArticle } = await withApi(() => import("./blog"));

    expect(await getPublishedArticle("inconnu")).toBeNull();
  });
});
