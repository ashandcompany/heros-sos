import { expect, test } from "@playwright/test";

for (const path of ["/", "/blog", "/encyclopedie", "/carte", "/faq"]) {
  test(`la page publique ${path} s'affiche`, async ({ page }) => {
    const response = await page.goto(path);
    expect(response?.status()).toBe(200);
    await expect(page.getByRole("heading", { level: 1 })).toBeVisible();
  });
}

test("robots.txt exclut les espaces connectés et pointe vers le sitemap", async ({ request }) => {
  const body = await (await request.get("/robots.txt")).text();
  expect(body).toContain("Disallow: /admin");
  expect(body).toContain("Sitemap: https://heros-sos.test/sitemap.xml");
});

test("le sitemap liste les articles et les fiches avec SITE_URL", async ({ request }) => {
  const response = await request.get("/sitemap.xml");
  expect(response.ok()).toBe(true);
  const body = await response.text();
  expect(body).toContain("<loc>https://heros-sos.test/blog/");
  expect(body).toContain("<loc>https://heros-sos.test/encyclopedie/super-heros/");
  expect(body).toContain("<loc>https://heros-sos.test/encyclopedie/super-vilains/");
});

test("un article a son aperçu Open Graph", async ({ page, request }) => {
  await page.goto("/blog");
  await page.getByRole("article").first().getByRole("link").click();
  await expect(page).toHaveURL(/\/blog\/.+/);

  const image = await page.locator('meta[property="og:image"]').getAttribute("content");
  expect(image).toMatch(/^https:\/\/heros-sos\.test\/blog\/.+\/opengraph-image/);

  const response = await request.get(new URL(image!).pathname);
  expect(response.headers()["content-type"]).toBe("image/png");
});
