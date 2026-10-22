import { expect, test } from "@playwright/test";
import { accounts, login } from "./helpers";

test("un modérateur valide un incident en attente", async ({ page }) => {
  await login(page, accounts.admin);
  await expect(page).toHaveURL(/\/admin$/);

  // Par défaut, la file ne montre que les incidents en attente.
  await page.goto("/admin/incidents");
  const pending = page.getByRole("article");
  await expect(pending.first()).toBeVisible();

  // Le premier en attente, quel qu'il soit : d'autres parcours ont pu en ajouter ou en traiter.
  const title = (await pending.first().getByRole("heading").textContent())!.trim();
  await pending
    .first()
    .getByRole("button", { name: `Valider : ${title}` })
    .click();

  await expect(page.getByRole("heading", { name: title, exact: true })).toHaveCount(0);

  // Il apparaît dans les incidents validés.
  await page.goto("/admin/incidents?statut=valide");
  await expect(page.getByRole("heading", { name: title, exact: true })).toBeVisible();
});
