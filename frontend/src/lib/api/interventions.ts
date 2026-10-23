import type {
  HeroIntervention,
  InterventionOutcome,
  InterventionRef,
  InterventionTeam,
  MutationResult,
  OutcomeVote,
  TeamParticipation,
} from "@/types";
import { apiFailureMessage } from "@/lib/validation/form-state";
import { isApiEnabled, mockResponse } from "./config";
import { ApiError, type ApiSchema, apiFetch, getApiSession, toApiFailure } from "./http";

/*
 * Équipes de super-héros engagées sur un incident (`/interventions/*`). Le héros qui démarre
 * l'intervention en est le lead ; d'autres la rejoignent d'eux-mêmes ou y sont invités. Un héros
 * seul clôt directement ; une équipe vote à l'unanimité sur le résultat proposé, et un refus renvoie
 * la décision à la modération (`POST /admin/interventions/{id}/decision`).
 *
 * Sans API, les incidents mockés n'ont qu'un héros par intervention : les actions d'équipe
 * répondent qu'elles demandent l'API.
 */

const participations: Record<
  ApiSchema<"InterventionParticipant">["participation"],
  TeamParticipation
> = { invited: "invite", confirmed: "confirme" };

const votes: Record<ApiSchema<"InterventionParticipant">["outcomeValidation"], OutcomeVote> = {
  pending: "en-attente",
  approved: "approuve",
  rejected: "refuse",
};

export const fromApiOutcome = (status: "succeeded" | "failed"): InterventionOutcome =>
  status === "succeeded" ? "reussi" : "echoue";

const toApiOutcome = (outcome: InterventionOutcome) =>
  outcome === "reussi" ? ("succeeded" as const) : ("failed" as const);

/** Équipe vue par le héros `meId` (`null` pour un modérateur : personne n'y est « moi »). */
export function fromApiTeam(
  intervention: ApiSchema<"Intervention">,
  meId: string | null,
): InterventionTeam {
  const members = intervention.participants
    .filter(({ isActive, participation }) => isActive || participation === "invited")
    .map((participant) => ({
      hero: { id: participant.heroId, alias: participant.alias || "Super-héros" },
      role: participant.role === "lead" ? ("lead" as const) : ("membre" as const),
      participation: participations[participant.participation],
      vote: votes[participant.outcomeValidation],
      isMe: participant.heroId === meId,
    }))
    // Le lead d'abord, puis les membres confirmés, puis les invités.
    .sort(
      (a, b) =>
        Number(b.role === "lead") - Number(a.role === "lead") ||
        Number(b.participation === "confirme") - Number(a.participation === "confirme") ||
        a.hero.alias.localeCompare(b.hero.alias, "fr"),
    );
  const proposer = members.find(({ hero }) => hero.id === intervention.proposedBy);

  return {
    id: intervention.id,
    startedAt: intervention.startedAt,
    members,
    proposedOutcome: intervention.proposedStatus
      ? fromApiOutcome(intervention.proposedStatus)
      : null,
    proposedBy: proposer?.hero ?? null,
    notes: intervention.notes ?? "",
  };
}

/** Identifiant du compte connecté (vraie session), pour reconnaître le héros dans une équipe. */
export async function sessionUserId(token: string): Promise<string | null> {
  return (await getApiSession(token))?.user.id ?? null;
}

/**
 * Équipe d'une intervention, `null` si elle n'existe plus.
 * API : `GET /interventions/{id}` (héros et modérateurs).
 */
export async function getInterventionTeam(
  token: string,
  interventionId: string,
  meId: string | null,
): Promise<InterventionTeam | null> {
  try {
    const response = await apiFetch(`/interventions/${encodeURIComponent(interventionId)}`, {
      sessionToken: token,
    });
    return fromApiTeam((await response.json()) as ApiSchema<"Intervention">, meId);
  } catch (error) {
    if (error instanceof ApiError && error.status === 404) return null;
    throw error;
  }
}

/**
 * Interventions terminées d'un incident, des plus récentes aux plus anciennes ; le héros affiché
 * est le lead. API : `GET /incidents/{incidentId}/interventions` (héros et modérateurs).
 */
export async function getPastInterventions(
  token: string,
  incidentId: string,
): Promise<HeroIntervention[]> {
  const response = await apiFetch(`/incidents/${encodeURIComponent(incidentId)}/interventions`, {
    sessionToken: token,
  });
  const { items } = (await response.json()) as { items: ApiSchema<"Intervention">[] };
  return items.flatMap((intervention) => {
    if (intervention.status === "in_progress") return [];
    const lead = intervention.participants.find(({ role }) => role === "lead");
    return {
      id: intervention.id,
      incidentId: intervention.incidentId,
      hero: { id: lead?.heroId ?? "", alias: lead?.alias || "Super-héros" },
      startedAt: intervention.startedAt,
      endedAt: intervention.endedAt ?? null,
      outcome: fromApiOutcome(intervention.status),
    };
  });
}

/** Messages des refus métier de `/interventions/*` (codes `intervention.Code…` du backend). */
const interventionErrors: Record<string, string> = {
  incident_not_available: "Cet incident ne peut plus recevoir d'intervention.",
  hero_not_approved: "Ce profil de super-héros n'est pas approuvé.",
  intervention_already_active: "Une intervention est déjà en cours sur cet incident.",
  hero_already_in_intervention: "Ce super-héros participe déjà à une autre intervention en cours.",
  intervention_participant_already_exists: "Ce super-héros fait déjà partie de l'équipe.",
  intervention_invitation_not_pending: "Aucune invitation n'est en attente.",
  intervention_outcome_not_pending: "Aucun résultat n'attend de validation.",
  intervention_outcome_already_proposed: "Un résultat est déjà soumis au vote de l'équipe.",
  intervention_outcome_already_validated: "Votre vote est déjà enregistré.",
  intervention_dispute_not_found:
    "Aucun désaccord à arbitrer : aucun membre de l'équipe n'a refusé le résultat proposé.",
};

/** Convertit un échec d'appel en message affichable (le 401 redirige déjà vers la connexion). */
export function toInterventionFailure(
  error: unknown,
  context: string,
  overrides: Record<string, string> = {},
): { ok: false; error: string } {
  if (error instanceof ApiError) {
    const message = overrides[error.code] ?? interventionErrors[error.code];
    if (message) return { ok: false, error: message };
    if (error.status === 404) return { ok: false, error: "Intervention introuvable." };
    if (error.code === "validation_failed") {
      return { ok: false, error: "Certaines informations sont invalides." };
    }
  }
  return { ok: false, error: apiFailureMessage(toApiFailure(error, context)) };
}

const teamsNeedApi: MutationResult = {
  ok: false,
  error: "Les équipes de super-héros ne sont disponibles qu'avec l'API.",
};

async function post(token: string, path: string, body: object | undefined, context: string) {
  try {
    await apiFetch(path, { method: "POST", sessionToken: token, body });
    return { ok: true } as const;
  } catch (error) {
    return toInterventionFailure(error, context);
  }
}

const interventionPath = ({ interventionId }: InterventionRef, suffix: string) =>
  `/interventions/${encodeURIComponent(interventionId)}${suffix}`;

/** Le héros connecté rejoint l'équipe, sans invitation. API : `POST /interventions/{id}/join`. */
export async function joinIntervention(
  token: string,
  ref: InterventionRef,
): Promise<MutationResult> {
  if (!isApiEnabled) return mockResponse(teamsNeedApi);
  return post(token, interventionPath(ref, "/join"), undefined, "Rejoindre une intervention");
}

/**
 * Invite des super-héros, qui devront confirmer leur présence (un mail les prévient).
 * API : `POST /interventions/{id}/invitations` avec `{ heroIds }`.
 */
export async function inviteHeroes(
  token: string,
  ref: InterventionRef,
  heroIds: string[],
): Promise<MutationResult> {
  if (!isApiEnabled) return mockResponse(teamsNeedApi);
  try {
    await apiFetch(interventionPath(ref, "/invitations"), {
      method: "POST",
      sessionToken: token,
      body: { heroIds } satisfies ApiSchema<"InviteHeroesPayload">,
    });
    return { ok: true };
  } catch (error) {
    return toInterventionFailure(error, "Invitation de super-héros", {
      intervention_participant_already_exists:
        "Un des super-héros choisis fait déjà partie de l'équipe.",
      hero_not_approved: "Un des super-héros choisis n'est pas approuvé.",
    });
  }
}

/**
 * Accepte ou refuse l'invitation du héros connecté.
 * API : `POST /interventions/{id}/invitations/me/confirm` ou `…/decline`.
 */
export async function respondToInvitation(
  token: string,
  ref: InterventionRef,
  accept: boolean,
): Promise<MutationResult> {
  if (!isApiEnabled) return mockResponse(teamsNeedApi);
  return post(
    token,
    interventionPath(ref, `/invitations/me/${accept ? "confirm" : "decline"}`),
    undefined,
    "Réponse à une invitation",
  );
}

/**
 * Vote du héros connecté sur le résultat proposé. L'unanimité clôt l'intervention ; un refus
 * annule la proposition et prévient la modération.
 * API : `POST /interventions/{id}/outcome/validate` avec `{ approved }`.
 */
export async function voteOnOutcome(
  token: string,
  ref: InterventionRef,
  approved: boolean,
): Promise<MutationResult> {
  if (!isApiEnabled) return mockResponse(teamsNeedApi);
  return post(
    token,
    interventionPath(ref, "/outcome/validate"),
    { approved } satisfies ApiSchema<"InterventionOutcomeValidationPayload">,
    "Vote sur le résultat d'une intervention",
  );
}

/**
 * Résultat déclaré par un membre confirmé : clôt l'intervention d'un héros seul, ou la soumet au
 * vote de l'équipe. API : `POST /interventions/{id}/outcome`, puis `GET /interventions/{id}` pour
 * savoir si elle est close.
 */
export async function proposeOutcome(
  token: string,
  ref: InterventionRef,
  outcome: InterventionOutcome,
  notes: string,
): Promise<{ ok: true; closed: boolean } | { ok: false; error: string }> {
  try {
    await apiFetch(interventionPath(ref, "/outcome"), {
      method: "POST",
      sessionToken: token,
      body: {
        status: toApiOutcome(outcome),
        notes: notes || undefined,
      } satisfies ApiSchema<"InterventionOutcomePayload">,
    });
  } catch (error) {
    return toInterventionFailure(error, "Résultat d'une intervention", {
      intervention_participant_required: "Vous ne faites pas partie de l'équipe.",
    });
  }
  const response = await apiFetch(interventionPath(ref, ""), { sessionToken: token });
  const intervention = (await response.json()) as ApiSchema<"Intervention">;
  return { ok: true, closed: intervention.status !== "in_progress" };
}

/**
 * Arbitrage d'un désaccord par la modération : tranche le résultat et clôt l'intervention.
 * L'API ne dit pas d'avance s'il y a un désaccord : elle répond 409 sinon.
 * API : `POST /admin/interventions/{id}/decision` avec `{ status, notes }`.
 */
export async function arbitrateIntervention(
  token: string,
  ref: InterventionRef,
  outcome: InterventionOutcome,
  notes: string,
): Promise<MutationResult> {
  if (!isApiEnabled) return mockResponse(teamsNeedApi);
  return post(
    token,
    `/admin/interventions/${encodeURIComponent(ref.interventionId)}/decision`,
    {
      status: toApiOutcome(outcome),
      notes: notes || undefined,
    } satisfies ApiSchema<"InterventionOutcomePayload">,
    "Arbitrage d'une intervention",
  );
}
