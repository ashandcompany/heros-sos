import type { MetadataRoute } from "next";
import { connection } from "next/server";
import { getPublishedArticles, getSuperHeroes, getVillains } from "@/lib/api";
import { blogArticleRoute, heroSheetRoute, routes, villainSheetRoute } from "@/lib/routes";
import { getSiteUrl } from "@/lib/site";

const publicPages = [
  routes.home,
  routes.encyclopedia,
  routes.map,
  routes.videos,
  routes.blog,
  routes.about,
  routes.contact,
  routes.faq,
  routes.legal,
  routes.privacy,
];

/** Une liste indisponible (API injoignable) est omise plutôt que de faire échouer tout le sitemap. */
async function orEmpty<T>(read: () => Promise<T[]>, context: string): Promise<T[]> {
  try {
    return await read();
  } catch (error) {
    console.error(`Sitemap : ${context} indisponibles`, error);
    return [];
  }
}

/** Pages publiques, articles publiés et fiches de l'encyclopédie, lus à chaque requête. */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  // SITE_URL et les données (API ou mocks) sont lues au démarrage du serveur, pas au build.
  await connection();
  const siteUrl = getSiteUrl();

  const [articles, heroes, villains] = await Promise.all([
    orEmpty(getPublishedArticles, "articles"),
    orEmpty(getSuperHeroes, "super-héros"),
    orEmpty(getVillains, "super-vilains"),
  ]);

  return [
    ...publicPages.map((path) => ({ url: `${siteUrl}${path === "/" ? "" : path}` })),
    ...articles.map((article) => ({
      url: `${siteUrl}${blogArticleRoute(article.slug)}`,
      lastModified: article.updatedAt,
    })),
    ...heroes.map((hero) => ({ url: `${siteUrl}${heroSheetRoute(hero.id)}` })),
    ...villains.map((villain) => ({ url: `${siteUrl}${villainSheetRoute(villain.id)}` })),
  ];
}
