"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import {
  createArticle,
  deleteArticle,
  createIncidentType,
  deleteVideo,
  getIncidentTypes,
  getSuperHeroes,
  getVillains,
  moderateAccount,
  moderateComment,
  moderateIncident,
  moderateVideo,
  reviewEmailChange,
  reviewHeroApplication,
  updateArticle,
  updateIncident,
} from "@/lib/api";
import { isModerationStatus } from "@/lib/format";
import { adminArticleRoute, adminIncidentRoute, routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import {
  readArticleValues,
  readIncidentEditValues,
  readIncidentTypeValues,
  toIncidentTypeInput,
  toArticleInput,
  toIncidentUpdate,
  validateArticle,
  validateIncidentEdit,
  validateIncidentType,
  type ArticleField,
  type IncidentEditField,
  type IncidentTypeField,
} from "@/lib/validation/admin";
import { apiFailureMessage, hasFieldErrors, type FormState } from "@/lib/validation/form-state";
import { validateRejectionReason } from "@/lib/validation/hero";
import type {
  AccountModerationAction,
  AccountModerationResult,
  HeroReviewResult,
  ModerationStatus,
  MutationResult,
} from "@/types";

/*
 * Server Actions du back-office. Chacune revérifie la session admin (le proxy ne suffit pas)
 * et l'API Go refera ce contrôle de son côté.
 */

const invalidRequest: FormState = {
  status: "error",
  message: "Requête invalide.",
  fieldErrors: {},
};

/** Applique le résultat d'une mutation : message, puis rafraîchit les pages concernées. */
function settle(result: MutationResult, paths: string[], message: string): FormState {
  if (!result.ok) return { status: "error", message: result.error, fieldErrors: {} };
  for (const path of [routes.admin, ...paths]) revalidatePath(path);
  return { status: "success", message, fieldErrors: {} };
}

/** Lit l'identifiant et la décision (`valide`, `rejete`…) envoyés par un formulaire de modération. */
function readDecision(formData: FormData): { id: string; decision: ModerationStatus } | null {
  const id = String(formData.get("id") ?? "");
  const decision = String(formData.get("decision") ?? "");
  return id && isModerationStatus(decision) ? { id, decision } : null;
}

const decisionMessages: Record<ModerationStatus, string> = {
  "en-attente": "Remis en attente de vérification.",
  valide: "Validé.",
  rejete: "Rejeté.",
};

/* ---------- Incidents ---------- */

export async function moderateIncidentAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const request = readDecision(formData);
  if (!request) return invalidRequest;

  const result = await moderateIncident(token, request.id, request.decision);
  return settle(
    result,
    [routes.adminIncidents, adminIncidentRoute(request.id), routes.home, routes.profileIncidents],
    {
      "en-attente": "Incident remis en attente de vérification.",
      valide: "Incident validé : la déclaration est conforme.",
      rejete: "Incident rejeté : il n'est plus diffusé.",
    }[request.decision],
  );
}

export async function updateIncidentAction(
  _previousState: FormState<IncidentEditField>,
  formData: FormData,
): Promise<FormState<IncidentEditField>> {
  const token = await requireAdminSession();
  const id = String(formData.get("id") ?? "");
  const values = readIncidentEditValues(formData);

  const [villains, heroes, types] = await Promise.all([
    getVillains(),
    getSuperHeroes(),
    getIncidentTypes(token),
  ]);
  const fieldErrors = validateIncidentEdit(values, {
    villainIds: villains.map(({ id: villainId }) => villainId),
    heroIds: heroes.map(({ id: heroId }) => heroId),
    typeIds: types.map(({ id: typeId }) => typeId),
  });
  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await updateIncident(token, id, toIncidentUpdate(values));
  if (!result.ok) return { status: "error", message: result.error, fieldErrors: {}, values };

  for (const path of [
    routes.admin,
    routes.adminIncidents,
    adminIncidentRoute(id),
    routes.home,
    routes.profileIncidents,
  ]) {
    revalidatePath(path);
  }
  return { status: "success", message: "Les modifications ont été enregistrées.", fieldErrors: {} };
}

/* ---------- Types d'incident ---------- */

export async function createIncidentTypeAction(
  _previousState: FormState<IncidentTypeField>,
  formData: FormData,
): Promise<FormState<IncidentTypeField>> {
  const token = await requireAdminSession();
  const values = readIncidentTypeValues(formData);
  const fieldErrors = validateIncidentType(values);
  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await createIncidentType(token, toIncidentTypeInput(values));
  if (!result.ok) {
    switch (result.error) {
      case "code-already-used":
        return { status: "error", fieldErrors: { code: "Ce code est déjà utilisé." }, values };
      case "invalid-fields":
        return {
          status: "error",
          fieldErrors: {
            code: result.fields.code && "Ce code n'a pas été accepté.",
            label: result.fields.label && "Ce libellé n'a pas été accepté.",
          },
          values,
        };
      default:
        return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
    }
  }

  // Les formulaires de déclaration proposent aussitôt le nouveau type.
  for (const path of [routes.adminIncidentTypes, routes.profileIncidents, routes.heroDeclare]) {
    revalidatePath(path);
  }
  return {
    status: "success",
    message: `Type « ${result.type.label} » créé : il est proposé dans les formulaires de déclaration.`,
    fieldErrors: {},
  };
}

/* ---------- Comptes ---------- */

export async function moderateAccountAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const id = String(formData.get("id") ?? "");
  const action = String(formData.get("decision") ?? "");
  if (!id || !isAccountModerationAction(action)) return invalidRequest;

  const result = await moderateAccount(token, id, action);
  if (!result.ok) {
    return { status: "error", message: accountModerationErrorMessage(result), fieldErrors: {} };
  }

  for (const path of [routes.admin, routes.adminAccounts]) revalidatePath(path);
  return { status: "success", message: accountModerationMessages[action], fieldErrors: {} };
}

const accountModerationMessages: Record<AccountModerationAction, string> = {
  deactivate: "Compte désactivé : la personne est déconnectée et ne peut plus se connecter.",
  reactivate: "Compte réactivé : la personne peut de nouveau se connecter.",
  verify: "Compte marqué comme vérifié.",
};

const isAccountModerationAction = (value: string): value is AccountModerationAction =>
  value in accountModerationMessages;

function accountModerationErrorMessage(
  result: Extract<AccountModerationResult, { ok: false }>,
): string {
  switch (result.error) {
    case "not-found":
      return "Ce compte n'existe plus.";
    case "self-action":
      return "Vous ne pouvez pas effectuer cette action sur votre propre compte.";
    case "protected-account":
      return "Un compte de modérateur ne peut pas être désactivé depuis le back-office.";
    default:
      return apiFailureMessage(result);
  }
}

export async function reviewEmailChangeAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const request = readDecision(formData);
  if (!request || request.decision === "en-attente") return invalidRequest;

  const approve = request.decision === "valide";
  const result = await reviewEmailChange(token, request.id, approve);
  return settle(
    result,
    [routes.adminAccounts],
    approve ? "Nouvelle adresse email activée." : "Changement d'adresse email refusé.",
  );
}

/* ---------- Super-héros ---------- */

function heroReviewError(
  result: Extract<HeroReviewResult, { ok: false }>,
  reason: string,
): FormState<"reason"> {
  const values = { reason };
  switch (result.error) {
    case "already-reviewed":
      return {
        status: "error",
        message: "Cette demande a déjà été traitée.",
        fieldErrors: {},
        values,
      };
    case "not-found":
      return { status: "error", message: "Cette demande n'existe plus.", fieldErrors: {}, values };
    case "invalid-reason":
      return {
        status: "error",
        fieldErrors: { reason: "Ce motif n'a pas été accepté : vérifiez sa longueur." },
        values,
      };
    default:
      return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
  }
}

/**
 * Accepte ou refuse une demande de statut super-héros. Le motif est obligatoire pour un refus :
 * il est communiqué au demandeur.
 */
export async function reviewHeroApplicationAction(
  _previousState: FormState<"reason">,
  formData: FormData,
): Promise<FormState<"reason">> {
  const token = await requireAdminSession();
  const request = readDecision(formData);
  if (!request || request.decision === "en-attente") return invalidRequest;

  const approve = request.decision === "valide";
  const reason = String(formData.get("reason") ?? "").trim();
  if (!approve) {
    const reasonError = validateRejectionReason(reason);
    if (reasonError) {
      return { status: "error", fieldErrors: { reason: reasonError }, values: { reason } };
    }
  }

  const result = await reviewHeroApplication(
    token,
    request.id,
    approve ? { approve: true } : { approve: false, reason },
  );
  if (!result.ok) return heroReviewError(result, reason);

  for (const path of [routes.admin, routes.adminHeroes, routes.home, routes.encyclopedia]) {
    revalidatePath(path);
  }
  return {
    status: "success",
    message: approve
      ? "Demande acceptée : le compte a maintenant accès à l'espace super-héros."
      : "Demande refusée : le motif a été communiqué au demandeur.",
    fieldErrors: {},
  };
}

/* ---------- Vidéothèque ---------- */

export async function moderateVideoAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const request = readDecision(formData);
  if (!request) return invalidRequest;

  const result = await moderateVideo(token, request.id, request.decision);
  return settle(result, [routes.adminVideos], decisionMessages[request.decision]);
}

export async function deleteVideoAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const id = String(formData.get("id") ?? "");
  if (!id) return invalidRequest;

  const result = await deleteVideo(token, id);
  return settle(result, [routes.adminVideos], "Film supprimé.");
}

/* ---------- Commentaires ---------- */

export async function moderateCommentAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const request = readDecision(formData);
  if (!request) return invalidRequest;

  const result = await moderateComment(token, request.id, request.decision);
  return settle(
    result,
    [routes.adminComments],
    {
      "en-attente": "Commentaire remis en attente.",
      valide: "Commentaire publié.",
      rejete: "Commentaire masqué (la note reste prise en compte).",
    }[request.decision],
  );
}

/* ---------- Blog ---------- */

export async function saveArticleAction(
  _previousState: FormState<ArticleField>,
  formData: FormData,
): Promise<FormState<ArticleField>> {
  const token = await requireAdminSession();
  const id = String(formData.get("id") ?? "");
  const values = readArticleValues(formData);
  const fieldErrors = validateArticle(values);

  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const input = toArticleInput(values);
  if (id) {
    const result = await updateArticle(token, id, input);
    if (!result.ok) return { status: "error", message: result.error, fieldErrors: {}, values };
    for (const path of [routes.admin, routes.adminBlog, adminArticleRoute(id), routes.blog]) {
      revalidatePath(path);
    }
    return { status: "success", message: "Article enregistré.", fieldErrors: {} };
  }

  const result = await createArticle(token, input);
  if (!result.ok || !result.id) {
    return {
      status: "error",
      message: result.ok ? "L'article n'a pas pu être créé." : result.error,
      fieldErrors: {},
      values,
    };
  }
  for (const path of [routes.admin, routes.adminBlog, routes.blog]) revalidatePath(path);
  redirect(adminArticleRoute(result.id));
}

export async function deleteArticleAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const id = String(formData.get("id") ?? "");
  if (!id) return invalidRequest;

  const result = await deleteArticle(token, id);
  if (!result.ok) return { status: "error", message: result.error, fieldErrors: {} };

  for (const path of [routes.admin, routes.adminBlog, routes.blog]) revalidatePath(path);
  redirect(routes.adminBlog);
}
