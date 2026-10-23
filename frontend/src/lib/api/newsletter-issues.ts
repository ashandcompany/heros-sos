import { citizensMock } from "@/mocks/citizens";
import { newslettersMock } from "@/mocks/newsletters";
import type {
  MutationResult,
  NewsletterIssue,
  NewsletterIssueInput,
  NewsletterIssueSummary,
  NewsletterSendResult,
  Page,
  PageRequest,
} from "@/types";
import { apiFailureMessage } from "@/lib/validation/form-state";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { fromApiPage, pageQuery, toMockPage } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Numéros de la newsletter (back-office, `/admin/newsletters`, réservé aux modérateurs) : brouillons
 * modifiables, puis envoi unique aux abonnés confirmés. Un numéro envoyé est en lecture seule.
 * Les listes ne contiennent pas le corps.
 */

/* ---------- Conversion depuis l'API ---------- */

function fromApiSummary(issue: ApiSchema<"NewsletterSummary">): NewsletterIssueSummary {
  return {
    id: issue.id,
    subject: issue.subject,
    recipientUserId: issue.recipientUserId ?? null,
    createdAt: issue.createdAt,
    sentAt: issue.sentAt ?? null,
  };
}

function fromApiIssue(issue: ApiSchema<"Newsletter">): NewsletterIssue {
  return { ...fromApiSummary(issue), body: issue.body };
}

function toIssueFailure(error: unknown, context: string): { ok: false; error: string } {
  if (error instanceof ApiError) {
    if (error.status === 404) return { ok: false, error: "Newsletter introuvable." };
    if (error.code === "newsletter_already_sent") {
      return {
        ok: false,
        error: "Cette newsletter a déjà été envoyée : elle ne peut plus changer.",
      };
    }
    if (error.code === "no_subscribed_recipient") {
      return {
        ok: false,
        error: "Aucun abonné confirmé ne recevrait cette newsletter : elle reste en brouillon.",
      };
    }
    if (error.code === "validation_failed") {
      return { ok: false, error: "Certaines informations de la newsletter sont invalides." };
    }
  }
  return { ok: false, error: apiFailureMessage(toApiFailure(error, context)) };
}

/* ---------- Mocks ---------- */

function toSummary({ body, ...summary }: NewsletterIssue): NewsletterIssueSummary {
  void body;
  return summary;
}

const findMock = (id: string) => newslettersMock.find((issue) => issue.id === id);

/* ---------- Back-office ---------- */

/**
 * Une page de numéros (brouillons et envoyés), les plus récents d'abord, sans leur corps.
 * API : `GET /admin/newsletters?limit=…&offset=…`.
 */
export async function getNewsletterIssues(
  token: string,
  page: PageRequest = {},
): Promise<Page<NewsletterIssueSummary>> {
  if (isApiEnabled) {
    const response = await apiFetch(`/admin/newsletters${pageQuery(page)}`, {
      sessionToken: token,
    });
    return fromApiPage(
      (await response.json()) as ApiSchema<"NewsletterSummaryPage">,
      fromApiSummary,
    );
  }

  await userFromToken(token, "admin");
  const sorted = [...newslettersMock].sort(
    (a, b) => Date.parse(b.createdAt) - Date.parse(a.createdAt),
  );
  return mockResponse(toMockPage(sorted.map(toSummary), page));
}

/** Un numéro, corps compris. API : `GET /admin/newsletters/{id}` (404 → `null`). */
export async function getNewsletterIssue(
  token: string,
  id: string,
): Promise<NewsletterIssue | null> {
  if (isApiEnabled) {
    try {
      const response = await apiFetch(`/admin/newsletters/${encodeURIComponent(id)}`, {
        sessionToken: token,
      });
      return fromApiIssue((await response.json()) as ApiSchema<"Newsletter">);
    } catch (error) {
      if (error instanceof ApiError && error.status === 404) return null;
      throw error;
    }
  }

  await userFromToken(token, "admin");
  return mockResponse(findMock(id) ?? null);
}

/**
 * Enregistre un brouillon ; l'API assainit le corps (balises de l'éditeur seulement).
 * API : `POST /admin/newsletters`.
 */
export async function createNewsletterIssue(
  token: string,
  input: NewsletterIssueInput,
): Promise<MutationResult & { id?: string }> {
  if (isApiEnabled) {
    try {
      const response = await apiFetch("/admin/newsletters", {
        method: "POST",
        sessionToken: token,
        body: input satisfies ApiSchema<"NewsletterPayload">,
      });
      return { ok: true, id: ((await response.json()) as ApiSchema<"Newsletter">).id };
    } catch (error) {
      return toIssueFailure(error, "Création d'une newsletter");
    }
  }

  await userFromToken(token, "admin");
  const id = `newsletter-${crypto.randomUUID()}`;
  newslettersMock.push({ id, ...input, createdAt: new Date().toISOString(), sentAt: null });
  return mockResponse({ ok: true, id });
}

/** Remplace un brouillon. API : `PUT /admin/newsletters/{id}` (envoyé : `newsletter_already_sent`). */
export async function updateNewsletterIssue(
  token: string,
  id: string,
  input: NewsletterIssueInput,
): Promise<MutationResult> {
  if (isApiEnabled) {
    try {
      await apiFetch(`/admin/newsletters/${encodeURIComponent(id)}`, {
        method: "PUT",
        sessionToken: token,
        body: input satisfies ApiSchema<"NewsletterPayload">,
      });
      return { ok: true };
    } catch (error) {
      return toIssueFailure(error, "Modification d'une newsletter");
    }
  }

  await userFromToken(token, "admin");
  const issue = findMock(id);
  if (!issue) return mockResponse({ ok: false, error: "Newsletter introuvable." });
  if (issue.sentAt) return toIssueFailure(new ApiError(409, "newsletter_already_sent"), "");
  Object.assign(issue, input);
  return mockResponse({ ok: true });
}

/** Supprime un brouillon. API : `DELETE /admin/newsletters/{id}` (envoyé : `newsletter_already_sent`). */
export async function deleteNewsletterIssue(token: string, id: string): Promise<MutationResult> {
  if (isApiEnabled) {
    try {
      await apiFetch(`/admin/newsletters/${encodeURIComponent(id)}`, {
        method: "DELETE",
        sessionToken: token,
      });
      return { ok: true };
    } catch (error) {
      return toIssueFailure(error, "Suppression d'une newsletter");
    }
  }

  await userFromToken(token, "admin");
  const index = newslettersMock.findIndex((issue) => issue.id === id);
  if (index === -1) return mockResponse({ ok: false, error: "Newsletter introuvable." });
  if (newslettersMock[index].sentAt) {
    return toIssueFailure(new ApiError(409, "newsletter_already_sent"), "");
  }
  newslettersMock.splice(index, 1);
  return mockResponse({ ok: true });
}

/**
 * Envoie un brouillon à ses abonnés confirmés, une seule fois. L'API répond dès que les mails sont
 * confiés à la file d'envoi (202), sans attendre leur remise.
 * API : `POST /admin/newsletters/{id}/send`.
 */
export async function sendNewsletterIssue(
  token: string,
  id: string,
): Promise<NewsletterSendResult> {
  if (isApiEnabled) {
    try {
      const response = await apiFetch(`/admin/newsletters/${encodeURIComponent(id)}/send`, {
        method: "POST",
        sessionToken: token,
      });
      const { recipients } = (await response.json()) as ApiSchema<"NewsletterSendResult">;
      return { ok: true, recipients };
    } catch (error) {
      return toIssueFailure(error, "Envoi d'une newsletter");
    }
  }

  await userFromToken(token, "admin");
  const issue = findMock(id);
  if (!issue) return mockResponse({ ok: false, error: "Newsletter introuvable." });
  if (issue.sentAt) return toIssueFailure(new ApiError(409, "newsletter_already_sent"), "");
  const recipients = citizensMock.filter(
    (citizen) =>
      citizen.newsletterSubscribed &&
      (!issue.recipientUserId || citizen.id === issue.recipientUserId),
  ).length;
  if (recipients === 0) return toIssueFailure(new ApiError(422, "no_subscribed_recipient"), "");
  issue.sentAt = new Date().toISOString();
  return mockResponse({ ok: true, recipients });
}
