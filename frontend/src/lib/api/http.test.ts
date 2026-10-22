import { describe, expect, it, vi } from "vitest";
import { errorResponse, jsonResponse, mockFetch, withApi } from "./test-utils";

const loadHttp = () => withApi(() => import("./http"));

describe("apiFetch", () => {
  it("transmet la session dans le cookie attendu par l'API, jamais dans Authorization", async () => {
    const fetchMock = mockFetch(() => jsonResponse({}));
    const { apiFetch } = await loadHttp();

    await apiFetch("/me", { sessionToken: "jeton" });

    const [url, init] = fetchMock.mock.calls[0] as unknown as [string, RequestInit];
    const headers = new Headers(init.headers);
    expect(url).toBe("http://api.test/api/v1/me");
    expect(headers.get("Cookie")).toBe("session=jeton");
    expect(headers.has("Authorization")).toBe(false);
  });

  it("lève une ApiError avec le code, les champs refusés et le délai Retry-After", async () => {
    mockFetch(() =>
      jsonResponse(
        { error: { code: "validation_failed", fields: { email: "invalid", age: 3 } } },
        { status: 422, headers: { "Retry-After": "30" } },
      ),
    );
    const { apiFetch } = await loadHttp();

    await expect(apiFetch("/auth/register")).rejects.toMatchObject({
      status: 422,
      code: "validation_failed",
      fields: { email: "invalid" },
      retryAfterSeconds: 30,
    });
  });

  it("donne le code `unknown` à une erreur sans enveloppe", async () => {
    mockFetch(() => new Response("Bad Gateway", { status: 502 }));
    const { apiFetch } = await loadHttp();

    await expect(apiFetch("/articles")).rejects.toMatchObject({ status: 502, code: "unknown" });
  });
});

describe("toApiFailure", () => {
  it.each([
    [403, "forbidden"],
    [413, "payload-too-large"],
    [415, "unsupported-media-type"],
    [500, "unavailable"],
  ])("traduit un %i en %s", async (status, expected) => {
    vi.spyOn(console, "warn").mockImplementation(() => {});
    vi.spyOn(console, "error").mockImplementation(() => {});
    mockFetch(() => errorResponse(status, "code"));
    const { apiFetch, toApiFailure } = await loadHttp();

    const error = await apiFetch("/x").catch((caught: unknown) => caught);
    expect(toApiFailure(error, "Test")).toEqual({ ok: false, error: expected });
  });

  it("garde le délai d'un 429", async () => {
    mockFetch(() => errorResponse(429, "rate_limited", { "Retry-After": "120" }));
    const { apiFetch, toApiFailure } = await loadHttp();

    const error = await apiFetch("/x").catch((caught: unknown) => caught);
    expect(toApiFailure(error, "Test")).toEqual({
      ok: false,
      error: "too-many-attempts",
      retryAfterSeconds: 120,
    });
  });

  it("considère une API injoignable comme indisponible", async () => {
    vi.spyOn(console, "error").mockImplementation(() => {});
    const { toApiFailure } = await loadHttp();

    expect(toApiFailure(new TypeError("fetch failed"), "Test")).toEqual({
      ok: false,
      error: "unavailable",
    });
  });
});

describe("fromApiRole", () => {
  it("traduit les rôles de l'API", async () => {
    const { fromApiRole } = await loadHttp();

    expect(fromApiRole("citizen")).toBe("citoyen");
    expect(fromApiRole("hero")).toBe("super-heros");
    expect(fromApiRole("moderator")).toBe("admin");
  });
});
