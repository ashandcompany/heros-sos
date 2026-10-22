import { defineConfig, devices } from "@playwright/test";

/**
 * Parcours dans le navigateur (`npm run test:e2e`), sur le build de production en mode mock.
 *
 * Les mocks vivent en mémoire dans un seul serveur et les parcours les modifient (inscription,
 * modération, prise en charge) : un seul worker, et des tests qui ne supposent pas un état neuf.
 * Port dédié, pour ne jamais réutiliser un `npm run dev` branché sur l'API par `.env.local`.
 */
const PORT = 3100;

export default defineConfig({
  testDir: "e2e",
  fullyParallel: false,
  workers: 1,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["html", { open: "never" }]] : "list",
  use: {
    baseURL: `http://localhost:${PORT}`,
    locale: "fr-FR",
    timezoneId: "Europe/Paris",
    trace: "on-first-retry",
  },
  projects: [{ name: "chromium", use: { ...devices["Desktop Chrome"] } }],
  webServer: {
    command: `npm run build && npm run start -- --port ${PORT}`,
    url: `http://localhost:${PORT}`,
    reuseExistingServer: !process.env.CI,
    timeout: 300_000,
    // Vide : prioritaire sur l'API_URL de .env.local, l'application tourne sur les mocks.
    env: { API_URL: "", SITE_URL: "https://heros-sos.test" },
  },
});
