import { defineConfig, devices } from "@playwright/test";

/**
 * Parcours contre la pile RÉELLE lancée par `make up` (proxy, front, API, PostgreSQL) : `npm run test:e2e:live`.
 *
 * Le `playwright.config.ts` habituel tourne sur les mocks (sans API) : il ne peut pas prouver ce qui passe par
 * l'API, comme la file hors ligne rejouée par `/api/sync`. Ces tests écrivent dans la base de développement
 * (un incident de test par passage) : ils ne sont pas lancés par la CI.
 */
export default defineConfig({
  testDir: "e2e-live",
  fullyParallel: false,
  workers: 1,
  retries: 0,
  timeout: 120_000,
  reporter: "list",
  use: {
    baseURL: process.env.LIVE_BASE_URL ?? "http://localhost:8088",
    locale: "fr-FR",
    timezoneId: "Europe/Paris",
    trace: "retain-on-failure",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
});
