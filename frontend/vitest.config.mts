import { defineConfig } from "vitest/config";

/**
 * Tests unitaires (`npm test`) : code pur de `src/lib/` (validation, formatage, conversions de
 * l'API). Les parcours dans le navigateur sont dans `e2e/` (Playwright, `npm run test:e2e`).
 */
export default defineConfig({
  // Alias `@/*` de tsconfig.json.
  resolve: { tsconfigPaths: true },
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
    // `vi.stubEnv` / `vi.stubGlobal` annulés après chaque test.
    unstubEnvs: true,
    unstubGlobals: true,
  },
});
