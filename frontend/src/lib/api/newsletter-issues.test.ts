import { describe, expect, it } from "vitest";
import type { ApiSchema } from "./http";
import { errorResponse, jsonResponse, mockFetch, withApi } from "./test-utils";

const summary: ApiSchema<"NewsletterSummary"> = {
  id: "8f1c2a4e-0000-4000-8000-000000000001",
  subject: "Le bilan de la rentrée",
  recipientUserId: null,
  sent: true,
  createdAt: "2026-10-01T08:00:00Z",
  sentAt: "2026-10-01T10:00:00Z",
};

describe("newsletter-issues (API)", () => {
  it("convertit une page de GET /admin/newsletters", async () => {
    const fetchMock = mockFetch(() =>
      jsonResponse({ items: [summary], total: 21, limit: 20, offset: 20 }),
    );
    const { getNewsletterIssues } = await withApi(() => import("./newsletter-issues"));

    const page = await getNewsletterIssues("jeton", { limit: 20, offset: 20 });

    expect(page).toEqual({
      items: [
        {
          id: summary.id,
          subject: summary.subject,
          recipientUserId: null,
          createdAt: summary.createdAt,
          sentAt: summary.sentAt,
        },
      ],
      total: 21,
      limit: 20,
      offset: 20,
    });
    expect(fetchMock.mock.calls[0][0]).toContain("/admin/newsletters?limit=20&offset=20");
  });

  it("renvoie null pour un numéro inconnu (404)", async () => {
    mockFetch(() => errorResponse(404, "not_found"));
    const { getNewsletterIssue } = await withApi(() => import("./newsletter-issues"));

    expect(await getNewsletterIssue("jeton", "inconnu")).toBeNull();
  });

  it("renvoie le nombre de destinataires de l'envoi", async () => {
    const fetchMock = mockFetch(() => jsonResponse({ recipients: 42 }, { status: 202 }));
    const { sendNewsletterIssue } = await withApi(() => import("./newsletter-issues"));

    expect(await sendNewsletterIssue("jeton", summary.id)).toEqual({ ok: true, recipients: 42 });
    expect(fetchMock.mock.calls.map(([url]) => String(url))).toContainEqual(
      expect.stringContaining(`/admin/newsletters/${summary.id}/send`),
    );
  });

  it("explique un envoi sans abonné ou déjà parti", async () => {
    const { sendNewsletterIssue } = await withApi(() => import("./newsletter-issues"));

    mockFetch(() => errorResponse(422, "no_subscribed_recipient"));
    expect(await sendNewsletterIssue("jeton", summary.id)).toEqual({
      ok: false,
      error: expect.stringContaining("Aucun abonné"),
    });

    mockFetch(() => errorResponse(409, "newsletter_already_sent"));
    expect(await sendNewsletterIssue("jeton", summary.id)).toEqual({
      ok: false,
      error: expect.stringContaining("déjà été envoyée"),
    });
  });
});
