import { expect, test, type APIRequestContext, type Page } from "@playwright/test";

/**
 * Le mode hors ligne dans un vrai navigateur (Chromium), contre la pile réelle : service worker, IndexedDB,
 * événement « online », route `/api/sync` et API Go.
 *
 * Le test crée son propre citoyen jetable (inscription, puis activation avec le lien lu dans Mailpit) : il ne
 * dépend d'aucun compte de la base. `LIVE_EMAIL` et `LIVE_PASSWORD` imposent un compte existant à la place.
 */
const BASE_URL = process.env.LIVE_BASE_URL ?? "http://localhost:8088";
const MAILPIT_URL = process.env.LIVE_MAILPIT_URL ?? "http://localhost:8025";

interface Account {
  email: string;
  password: string;
}

let account: Account;

/** Inscrit un citoyen puis active son compte avec le jeton du mail reçu dans Mailpit. */
async function createCitizen(request: APIRequestContext): Promise<Account> {
  const email = `e2e-live-${Date.now()}@heros-sos.test`;
  const password = "Motdepasse1!";
  const register = await request.post("/api/v1/auth/register", {
    headers: { origin: BASE_URL },
    data: {
      civility: "non-precise",
      firstName: "Test",
      lastName: "HorsLigne",
      birthDate: "1990-04-12",
      email,
      mobilePhone: "0612345678",
      address: { street: "12 rue de la Paix", postalCode: "75002", city: "Paris", country: "FR" },
      password,
    },
  });
  expect(
    register.status(),
    register.status() === 429
      ? "Inscriptions limitées par adresse IP : réessayez plus tard, ou imposez un compte existant avec LIVE_EMAIL et LIVE_PASSWORD."
      : "inscription du citoyen de test",
  ).toBe(201);

  let token = "";
  await expect
    .poll(
      async () => {
        const list = await (await request.get(`${MAILPIT_URL}/api/v1/messages?limit=50`)).json();
        const mail = (list.messages ?? []).find((m: { To: { Address: string }[] }) =>
          m.To.some((to) => to.Address === email),
        );
        if (!mail) return "";
        const body = await (await request.get(`${MAILPIT_URL}/api/v1/message/${mail.ID}`)).json();
        token = /token=([\w-]+)/.exec(body.Text ?? "")?.[1] ?? "";
        return token;
      },
      { message: "le mail d'activation n'est pas arrivé dans Mailpit", timeout: 30_000 },
    )
    .not.toBe("");

  const activate = await request.post("/api/v1/auth/activate", {
    headers: { origin: BASE_URL },
    data: { token },
  });
  expect(activate.ok()).toBe(true);
  return { email, password };
}

test.beforeAll(async ({ playwright }) => {
  if (process.env.LIVE_EMAIL && process.env.LIVE_PASSWORD) {
    account = { email: process.env.LIVE_EMAIL, password: process.env.LIVE_PASSWORD };
    return;
  }
  const request = await playwright.request.newContext({ baseURL: BASE_URL });
  try {
    account = await createCitizen(request);
  } finally {
    await request.dispose();
  }
});

async function login(page: Page) {
  await page.goto("/connexion");
  await page.getByLabel("Adresse email ou pseudo").fill(account.email);
  await page.getByLabel("Mot de passe", { exact: true }).fill(account.password);
  await page.getByRole("button", { name: "Se connecter" }).click();
  await expect(page).not.toHaveURL(/\/connexion/);
}

/** Attend que le service worker soit actif et contrôle la page (un rechargement après son activation). */
async function waitForServiceWorker(page: Page) {
  await page.evaluate(async () => {
    await navigator.serviceWorker.ready;
  });
  await page.reload();
  await expect
    .poll(() => page.evaluate(() => navigator.serviceWorker.controller !== null))
    .toBe(true);
}

test("une page déjà visitée se rouvre sans réseau", async ({ page, context }) => {
  await login(page);
  await page.goto("/profil");
  await waitForServiceWorker(page);
  const heading = page.getByRole("heading", { level: 1 }).first();
  const title = await heading.innerText();

  await context.setOffline(true);
  await page.reload();
  await expect(page.getByRole("heading", { level: 1 }).first()).toHaveText(title);

  await context.setOffline(false);
});

test("un incident déclaré hors ligne est envoyé au retour du réseau, une seule fois", async ({
  page,
  context,
}) => {
  await login(page);
  await page.goto("/profil/incidents");
  await waitForServiceWorker(page);

  const title = `[TEST hors ligne] ${Date.now()}`;
  const form = page.getByRole("region", { name: "Déclarer un incident" });
  await context.setOffline(true);
  // Le navigateur doit avoir pris acte de la coupure avant la validation du formulaire.
  await expect.poll(() => page.evaluate(() => navigator.onLine)).toBe(false);

  await form.getByRole("radio", { name: "Témoin" }).check();
  await form.getByLabel("Type d'incident").selectOption({ index: 1 });
  await form.getByLabel("Titre").fill(title);
  await form.getByLabel("Description").fill("Incident déclaré sans réseau par le test hors ligne.");
  await form.getByRole("combobox", { name: "Lieu précis" }).fill("Place Bellecour");
  await form.getByLabel("Ville").fill("Lyon");
  await form.getByRole("button", { name: "Déclarer l'incident" }).click();

  // L'action est gardée sur l'appareil, avec le badge d'attente.
  const status = page.getByRole("status");
  await expect(status.getByText("Hors ligne", { exact: false })).toBeVisible();
  await expect(status.getByText("1 action en attente")).toBeVisible();

  // Le réseau revient : la file part seule, par `/api/sync`.
  const sync = page.waitForRequest((request) => request.url().endsWith("/api/sync"));
  await context.setOffline(false);
  const sent = await sync;
  await expect(status.getByText("1 action synchronisée")).toBeVisible({ timeout: 30_000 });
  await expect(status.getByText("action en attente")).toBeHidden();

  // Côté serveur : l'incident existe, une seule fois.
  const list = await page.request.get("/api/v1/incidents?limit=100");
  expect(list.ok()).toBe(true);
  expect((await list.text()).split(title).length - 1).toBe(1);

  // Rejouer la même commande (même clé d'idempotence) : réponse identique, aucun doublon.
  const replay = await page.request.post("/api/sync", {
    data: sent.postData() ?? "",
    headers: { "content-type": "application/json" },
  });
  expect(await replay.json()).toEqual({ ok: true, replayed: true });
  const after = await page.request.get("/api/v1/incidents?limit=100");
  expect((await after.text()).split(title).length - 1).toBe(1);
});
