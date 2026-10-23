import { describe, expect, it, vi } from "vitest";
import { getSiteDomain } from "./site";

describe("getSiteDomain", () => {
  it("rend le domaine de SITE_URL, sans port ni slash final", () => {
    vi.stubEnv("SITE_URL", "https://heros-sos.fr/");
    expect(getSiteDomain()).toBe("heros-sos.fr");

    vi.stubEnv("SITE_URL", "http://localhost:8088");
    expect(getSiteDomain()).toBe("localhost");
  });

  it("prend celui du développement quand SITE_URL est vide", () => {
    vi.stubEnv("SITE_URL", "");
    expect(getSiteDomain()).toBe("localhost");
  });

  it("rend undefined, sans lever, quand SITE_URL n'est pas une URL", () => {
    vi.stubEnv("SITE_URL", "pas une url");
    expect(getSiteDomain()).toBeUndefined();
  });
});
