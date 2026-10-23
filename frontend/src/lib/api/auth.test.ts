import { describe, expect, it } from "vitest";
import { SESSION_COOKIE } from "@/lib/cookies";
import { errorResponse, jsonResponse, mockFetch, withApi } from "./test-utils";

/** Les écritures relisent la session (`GET /auth/me`) pour son jeton CSRF. */
const session = () => jsonResponse({ csrfToken: "csrf-1" });

describe("changePassword (API)", () => {
  it("renvoie le nouveau jeton de session posé par l'API", async () => {
    const fetchMock = mockFetch((path) =>
      path === "/auth/me"
        ? session()
        : jsonResponse(
            { expiresAt: "2026-10-30T10:00:00Z" },
            { headers: { "Set-Cookie": `${SESSION_COOKIE}=nouveau-jeton; Path=/; HttpOnly` } },
          ),
    );
    const { changePassword } = await withApi(() => import("./auth"));

    expect(await changePassword("token", "Ancien1234abc", "Nouveau5678xyz")).toEqual({
      ok: true,
      token: "nouveau-jeton",
      expiresAt: "2026-10-30T10:00:00Z",
    });
    const [, init] = fetchMock.mock.calls.find(([url]) =>
      String(url).endsWith("/me/password"),
    ) as unknown as [string, RequestInit];
    expect(JSON.parse(String(init.body))).toEqual({
      currentPassword: "Ancien1234abc",
      newPassword: "Nouveau5678xyz",
    });
  });

  it("distingue un mot de passe actuel faux d'un nouveau mot de passe trop faible", async () => {
    let field = "currentPassword";
    mockFetch((path) =>
      path === "/auth/me"
        ? session()
        : jsonResponse(
            { error: { code: "validation_failed", message: "", fields: { [field]: "invalid" } } },
            { status: 422 },
          ),
    );
    const { changePassword } = await withApi(() => import("./auth"));

    expect(await changePassword("token", "faux", "Nouveau5678xyz")).toEqual({
      ok: false,
      error: "invalid-current-password",
    });
    field = "newPassword";
    expect(await changePassword("token", "Ancien1234abc", "court")).toEqual({
      ok: false,
      error: "weak-password",
    });
  });

  it("rapporte une limite de débit", async () => {
    mockFetch((path) =>
      path === "/auth/me" ? session() : errorResponse(429, "rate_limited", { "Retry-After": "60" }),
    );
    const { changePassword } = await withApi(() => import("./auth"));

    expect(await changePassword("token", "Ancien1234abc", "Nouveau5678xyz")).toMatchObject({
      ok: false,
      error: "too-many-attempts",
    });
  });
});
