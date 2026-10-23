import { expect, type Page } from "@playwright/test";

/** Comptes mock (src/mocks/users.ts). */
export const accounts = {
  citizen: { email: "citoyen@heros-sos.fr", password: "Citoyen123!" },
  hero: { email: "lumen@heros-sos.fr", password: "Heros123!" },
  admin: { email: "admin@heros-sos.fr", password: "Admin123!" },
};

/** Connexion par le formulaire ; attend la redirection vers l'espace du rôle. */
export async function login(page: Page, { email, password }: { email: string; password: string }) {
  await page.goto("/connexion");
  await page.getByLabel("Adresse email ou pseudo").fill(email);
  await page.getByLabel("Mot de passe", { exact: true }).fill(password);
  await page.getByRole("button", { name: "Se connecter" }).click();
  await expect(page).not.toHaveURL(/\/connexion/);
}
