import { describe, expect, it } from "vitest";
import type { ApiSchema } from "./http";
import { errorResponse, jsonResponse, mockFetch, withApi } from "./test-utils";

const account: ApiSchema<"Account"> = {
  id: "hero-1",
  role: "hero",
  civility: "monsieur",
  firstName: "Tony",
  lastName: "Stark",
  birthDate: "1970-05-29",
  email: "tony@stark.io",
  mobilePhone: "0612345678",
  address: { street: "10880 Malibu Point", postalCode: "90265", city: "Malibu", country: "US" },
  newsletterSubscribed: false,
  createdAt: "2026-01-01T00:00:00Z",
  hero: { alias: "Iron Man", bio: "Armure." },
};

/** Les écritures relisent la session (`GET /auth/me`) pour son jeton CSRF. */
const session = () => jsonResponse({ csrfToken: "csrf-1" });

/** Appels à l'API hors lecture de la session : [chemin, méthode, corps JSON]. */
function writes(fetchMock: ReturnType<typeof mockFetch>) {
  return fetchMock.mock.calls
    .map((call) => call as unknown as [string, RequestInit])
    .filter(([url]) => !url.endsWith("/auth/me"))
    .map(([url, init]) => [
      new URL(url).pathname.replace("/api/v1", ""),
      init.method,
      init.body === undefined ? undefined : JSON.parse(String(init.body)),
      new Headers(init.headers).get("X-CSRF-Token"),
    ]);
}

describe("getMyHeroProfile (API)", () => {
  it("lit la date du consentement au partage de position", async () => {
    mockFetch(() =>
      jsonResponse({
        ...account,
        hero: { ...account.hero!, locationConsentAt: "2026-10-22T09:00:00Z" },
      }),
    );
    const { getMyHeroProfile } = await withApi(() => import("./super-heroes"));

    expect((await getMyHeroProfile("token")).locationConsentAt).toBe("2026-10-22T09:00:00Z");
  });

  it("donne null sans consentement", async () => {
    mockFetch(() => jsonResponse(account));
    const { getMyHeroProfile } = await withApi(() => import("./super-heroes"));

    expect((await getMyHeroProfile("token")).locationConsentAt).toBeNull();
  });
});

describe("partage de position (API)", () => {
  it("appelle chaque route avec la bonne méthode, le corps et le jeton CSRF", async () => {
    const fetchMock = mockFetch((path) =>
      path === "/auth/me" ? session() : new Response(null, { status: 204 }),
    );
    const api = await withApi(() => import("./super-heroes"));

    expect(await api.grantHeroLocationConsent("token")).toEqual({ ok: true });
    expect(
      await api.saveHeroLocation("token", { latitude: 45.76, longitude: 4.83, accuracyM: 25 }),
    ).toEqual({
      ok: true,
    });
    expect(await api.deleteHeroLocation("token")).toEqual({ ok: true });
    expect(await api.revokeHeroLocationConsent("token")).toEqual({ ok: true });

    expect(writes(fetchMock)).toEqual([
      ["/hero/location/consent", "PUT", undefined, "csrf-1"],
      ["/hero/location", "PUT", { latitude: 45.76, longitude: 4.83, accuracyM: 25 }, "csrf-1"],
      ["/hero/location", "DELETE", undefined, "csrf-1"],
      ["/hero/location/consent", "DELETE", undefined, "csrf-1"],
    ]);
  });

  it("traduit l'absence de consentement en message clair", async () => {
    mockFetch((path) =>
      path === "/auth/me" ? session() : errorResponse(409, "hero_location_consent_required"),
    );
    const { saveHeroLocation } = await withApi(() => import("./super-heroes"));

    expect(await saveHeroLocation("token", { latitude: 45.76, longitude: 4.83 })).toEqual({
      ok: false,
      error: "Activez d'abord le partage de position.",
    });
  });
});
