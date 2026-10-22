import { expect, test } from "@playwright/test";
import { accounts, login } from "./helpers";

test("un visiteur crée un compte citoyen", async ({ page }) => {
  // Le mock garde les comptes jusqu'au redémarrage du serveur : une adresse neuve à chaque essai.
  const email = `e2e-${Date.now()}@heros-sos.test`;
  const password = "Motdepasse1";

  await page.goto("/inscription");
  // Le pied de page a aussi un champ « Adresse email » (newsletter).
  const signup = page.getByRole("main");

  // Étape 1 : informations personnelles.
  await signup.getByRole("radio", { name: "Madame" }).check();
  await signup.getByLabel("Prénom").fill("Camille");
  await signup.getByLabel("Nom", { exact: true }).fill("Testeuse");
  await signup.getByLabel("Date de naissance").fill("1990-04-12");
  await signup.getByLabel("Adresse email").fill(email);
  await signup.getByRole("button", { name: "Continuer" }).click();

  // Étape 2 : coordonnées et sécurité.
  await expect(signup.getByRole("heading", { name: /Étape 2 sur 2/ })).toBeFocused();
  await signup.getByLabel("Numéro de portable").fill("06 12 34 56 78");
  // Champ d'autocomplétion : sa liste de suggestions porte un libellé voisin.
  await signup.getByRole("combobox", { name: "Numéro et nom de la rue" }).fill("12 rue de la Paix");
  await signup.getByLabel("Code postal").fill("75002");
  await signup.getByLabel("Ville").fill("Paris");
  await signup.getByLabel("Mot de passe", { exact: true }).fill(password);
  await signup.getByLabel("Confirmation du mot de passe").fill(password);
  await signup.getByRole("button", { name: "Créer mon compte" }).click();

  await expect(signup.getByText("Compte créé : vérifiez vos mails.")).toBeVisible();
  // On ne poursuit pas avec ce compte : `proxy.ts` est un bundle à part, avec sa propre copie
  // des mocks en mémoire, et ne connaît pas un compte créé après le démarrage du serveur.
});

test("un citoyen déclare un incident et le retrouve dans ses incidents", async ({ page }) => {
  await login(page, accounts.citizen);
  await expect(page).toHaveURL(/\/profil$/);
  await page.goto("/profil/incidents");

  const title = `Rayon gelant sur le marché ${Date.now()}`;
  const form = page.getByRole("region", { name: "Déclarer un incident" });
  await form.getByRole("radio", { name: "Témoin" }).check();
  await form.getByLabel("Type d'incident").selectOption({ index: 1 });
  await form.getByLabel("Titre").fill(title);
  await form
    .getByLabel("Description")
    .fill("Un individu en combinaison bleue gèle les étals avec un rayon.");
  await form.getByRole("combobox", { name: "Lieu précis" }).fill("Marché de la Croix-Rousse");
  await form.getByLabel("Ville").fill("Lyon");
  await form.getByRole("button", { name: "Déclarer l'incident" }).click();

  await expect(form.getByText("Incident déclaré.", { exact: false })).toBeVisible();
  await page.reload();
  // Aussi listé dans « Incidents en cours » : on vérifie la liste personnelle.
  const mine = page.getByRole("region", { name: "Mes déclarations" });
  await expect(mine.getByRole("link", { name: title })).toBeVisible();
});
