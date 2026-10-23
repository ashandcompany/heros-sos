import type { MetadataRoute } from "next";
import { connection } from "next/server";
import { getSiteUrl } from "@/lib/site";
import { routes } from "@/lib/routes";

/**
 * Espaces connectés et pages ouvertes depuis un mail (`?token=…`) : rien à indexer. Les layouts
 * des espaces posent aussi `robots: noindex`.
 */
const privatePaths = [
  routes.admin,
  routes.heroSpace,
  routes.profile,
  routes.login,
  routes.activation,
  routes.passwordReset,
  "/newsletter",
  "/changement-email",
  "/api",
];

export default async function robots(): Promise<MetadataRoute.Robots> {
  // SITE_URL est lue au démarrage du serveur, pas au build.
  await connection();
  const siteUrl = getSiteUrl();

  return {
    rules: { userAgent: "*", allow: "/", disallow: privatePaths },
    sitemap: `${siteUrl}/sitemap.xml`,
  };
}
