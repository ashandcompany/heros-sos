import { expect, test } from "@playwright/test";
import { accounts, login } from "./helpers";

test("un super-héros prend en charge un incident signalé", async ({ page }) => {
  await login(page, accounts.hero);
  await expect(page).toHaveURL(/\/heros\/dashboard$/);

  await page.goto("/heros/incidents");
  // Un incident encore libre, quel qu'il soit : d'autres essais ont pu en prendre en charge.
  const reported = page.getByRole("article").filter({ hasText: "Signalé" }).first();
  const link = reported.getByRole("heading").getByRole("link");
  const title = (await link.textContent())!.trim();
  await link.click();

  await expect(page.getByRole("heading", { level: 1, name: title })).toBeVisible();
  await page.getByRole("button", { name: "Prendre en charge" }).click();

  // L'intervention est ouverte : on peut maintenant en donner le résultat.
  await expect(page.getByRole("button", { name: "Réussi" })).toBeVisible();
  await expect(page.getByRole("button", { name: "Prendre en charge" })).toHaveCount(0);
});
