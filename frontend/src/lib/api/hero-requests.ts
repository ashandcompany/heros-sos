import { citizensMock } from "@/mocks/citizens";
import { heroApplicationsMock } from "@/mocks/hero-applications";
import { superHeroesMock, superHeroProfilesMock } from "@/mocks/super-heroes";
import { accountsMock } from "@/mocks/users";
import type {
  HeroApplication,
  HeroRequestInput,
  HeroRequestResult,
  HeroReviewResult,
  ModerationStatus,
  MyHeroRequest,
  Page,
  PageRequest,
  SuperHero,
} from "@/types";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, toApiFailure } from "./http";
import { fromApiPage, pageQuery, toMockPage } from "./pagination";
import { userFromToken } from "./session-user";

/*
 * Demandes de statut super-héros : un citoyen dépose sa demande depuis « Mon compte », un
 * modérateur l'accepte (le compte devient super-héros) ou la refuse avec un motif, que le citoyen
 * voit et après lequel il peut redéposer une demande.
 */

type ApiRequestStatus = ApiSchema<"HeroRequestStatus">["status"];

const statusFromApi: Record<ApiRequestStatus, ModerationStatus> = {
  pending: "en-attente",
  approved: "valide",
  rejected: "rejete",
};

const statusToApi: Record<ModerationStatus, ApiRequestStatus> = {
  "en-attente": "pending",
  valide: "approved",
  rejete: "rejected",
};

function fromApiApplicant(applicant: ApiSchema<"HeroApplicant">): HeroApplication {
  return {
    userId: applicant.userId,
    alias: applicant.alias,
    bio: applicant.bio,
    requestMessage: applicant.requestMessage,
    firstName: applicant.firstName,
    lastName: applicant.lastName,
    email: applicant.email,
    submittedAt: applicant.createdAt,
    status: statusFromApi[applicant.status],
  };
}

/* ---------- Espace citoyen ---------- */

/**
 * Demande du citoyen connecté, ou `null` s'il n'en a jamais déposé.
 * API : `GET /heroes/requests/me` (404 `hero_request_not_found` sans demande).
 */
export async function getMyHeroRequest(token: string): Promise<MyHeroRequest | null> {
  if (!isApiEnabled) {
    const user = await userFromToken(token, "citoyen");
    const record = heroApplicationsMock.find(({ userId }) => userId === user.id);
    return mockResponse(
      record
        ? { status: record.status, alias: record.alias, rejectionReason: record.rejectionReason }
        : null,
    );
  }

  try {
    const response = await apiFetch("/heroes/requests/me", { sessionToken: token });
    const body = (await response.json()) as ApiSchema<"HeroRequestStatus">;
    return {
      status: statusFromApi[body.status],
      alias: body.alias,
      rejectionReason: body.rejectionReason ?? null,
    };
  } catch (error) {
    if (error instanceof ApiError && error.code === "hero_request_not_found") return null;
    throw error;
  }
}

const requestErrorByCode = new Map([
  ["hero_request_pending", "already-pending"],
  ["already_hero", "already-hero"],
  ["alias_already_used", "alias-already-used"],
  ["hero_request_not_eligible", "not-eligible"],
] as const);

/**
 * Dépose une demande (ou en redépose une après un refus).
 * API : `POST /heroes/requests` avec `{ alias, bio, requestMessage }`.
 */
export async function requestHeroStatus(
  token: string,
  input: HeroRequestInput,
): Promise<HeroRequestResult> {
  if (!isApiEnabled) return requestHeroStatusWithMocks(token, input);

  try {
    await apiFetch("/heroes/requests", {
      method: "POST",
      sessionToken: token,
      body: input satisfies ApiSchema<"HeroRequestPayload">,
    });
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      const known = requestErrorByCode.get(error.code as never);
      if (known) return { ok: false, error: known };
      if (error.code === "validation_failed") {
        return { ok: false, error: "invalid-fields", fields: error.fields };
      }
    }
    return toApiFailure(error, "Demande de statut super-héros");
  }
}

async function requestHeroStatusWithMocks(
  token: string,
  input: HeroRequestInput,
): Promise<HeroRequestResult> {
  const user = await userFromToken(token, "citoyen");
  const existing = heroApplicationsMock.find(({ userId }) => userId === user.id);
  if (existing?.status === "en-attente")
    return mockResponse({ ok: false, error: "already-pending" });
  if (existing?.status === "valide") return mockResponse({ ok: false, error: "already-hero" });

  const alias = input.alias.toLowerCase();
  const aliasTaken =
    superHeroesMock.some((hero) => hero.alias.toLowerCase() === alias) ||
    heroApplicationsMock.some(
      (record) =>
        record.userId !== user.id &&
        record.status === "en-attente" &&
        record.alias.toLowerCase() === alias,
    );
  if (aliasTaken) return mockResponse({ ok: false, error: "alias-already-used" });

  const citizen = citizensMock.find(({ id }) => id === user.id);
  const record = {
    userId: user.id,
    ...input,
    firstName: citizen?.firstName ?? user.displayName,
    lastName: citizen?.lastName ?? "",
    email: citizen?.email ?? "",
    submittedAt: new Date().toISOString(),
    status: "en-attente" as const,
    rejectionReason: null,
  };
  if (existing) Object.assign(existing, record);
  else heroApplicationsMock.push(record);

  return mockResponse({ ok: true });
}

/* ---------- Back-office (rôle admin) ---------- */

/**
 * Demandes de statut super-héros, les plus anciennes d'abord (ordre de traitement).
 * API : `GET /admin/hero-requests?status=…&limit=…&offset=…`.
 */
export async function getHeroApplications(
  token: string,
  { status, ...page }: PageRequest & { status?: ModerationStatus } = {},
): Promise<Page<HeroApplication>> {
  if (!isApiEnabled) {
    await userFromToken(token, "admin");
    const applications: HeroApplication[] = heroApplicationsMock
      .filter((application) => !status || application.status === status)
      .sort((a, b) => Date.parse(a.submittedAt) - Date.parse(b.submittedAt));
    return mockResponse(toMockPage(applications, page));
  }

  const query = pageQuery(page, { status: status && statusToApi[status] });
  const response = await apiFetch(`/admin/hero-requests${query}`, { sessionToken: token });
  return fromApiPage((await response.json()) as ApiSchema<"HeroApplicantPage">, fromApiApplicant);
}

export type HeroReviewDecision = { approve: true } | { approve: false; reason: string };

/**
 * Accepte une demande (le demandeur devient super-héros, sans se reconnecter) ou la refuse avec
 * un motif, communiqué au demandeur.
 * API : `POST /admin/hero-requests/{userId}/approve`, ou `…/reject` avec `{ reason }`.
 */
export async function reviewHeroApplication(
  token: string,
  userId: string,
  decision: HeroReviewDecision,
): Promise<HeroReviewResult> {
  if (!isApiEnabled) return reviewHeroApplicationWithMocks(token, userId, decision);

  try {
    await apiFetch(
      `/admin/hero-requests/${encodeURIComponent(userId)}/${decision.approve ? "approve" : "reject"}`,
      {
        method: "POST",
        sessionToken: token,
        body: decision.approve
          ? undefined
          : ({ reason: decision.reason } satisfies ApiSchema<"RejectPayload">),
      },
    );
    return { ok: true };
  } catch (error) {
    if (error instanceof ApiError) {
      if (error.code === "hero_request_not_pending")
        return { ok: false, error: "already-reviewed" };
      if (error.code === "not_found") return { ok: false, error: "not-found" };
      if (error.code === "validation_failed") return { ok: false, error: "invalid-reason" };
    }
    return toApiFailure(error, "Décision sur une demande de super-héros");
  }
}

async function reviewHeroApplicationWithMocks(
  token: string,
  userId: string,
  decision: HeroReviewDecision,
): Promise<HeroReviewResult> {
  await userFromToken(token, "admin");
  const application = heroApplicationsMock.find((existing) => existing.userId === userId);
  if (!application) return mockResponse({ ok: false, error: "not-found" });
  if (application.status !== "en-attente") {
    return mockResponse({ ok: false, error: "already-reviewed" });
  }

  if (!decision.approve) {
    application.status = "rejete";
    application.rejectionReason = decision.reason;
    return mockResponse({ ok: true });
  }

  application.status = "valide";
  const hero: SuperHero = {
    id: `hero-${crypto.randomUUID()}`,
    alias: application.alias,
    avatarUrl: null,
    power: application.bio,
    averageRating: 0,
    interventionsCount: 0,
  };
  superHeroesMock.push(hero);
  superHeroProfilesMock.push({
    ...hero,
    firstName: application.firstName,
    lastName: application.lastName,
    civility: null,
    birthDate: null,
    email: application.email,
    pendingEmail: null,
    mobilePhone: "",
    address: null,
    registeredAt: new Date().toISOString(),
    lastInterventionAt: null,
    accountStatus: "actif",
  });
  accountsMock.push({
    user: { id: hero.id, displayName: application.alias, role: "super-heros" },
    email: application.email,
    // Mock : mot de passe aléatoire, inutilisable (le compte mock ne sert qu'à l'annuaire).
    password: crypto.randomUUID(),
  });

  return mockResponse({ ok: true });
}
