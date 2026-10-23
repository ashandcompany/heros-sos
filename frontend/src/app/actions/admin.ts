"use server";

import { revalidatePath } from "next/cache";
import { redirect } from "next/navigation";
import {
  addHeroVideo,
  addVillainFilm,
  arbitrateIntervention,
  createArticle,
  deleteArticle,
  createIncidentType,
  createVillain,
  deleteVideo,
  deleteVillainFilm,
  getIncidentTypes,
  getSuperHeroes,
  getVillains,
  moderateAccount,
  moderateReply,
  moderateReview,
  moderateIncident,
  moderateVideo,
  parseHeroVideoKey,
  reviewEmailChange,
  reviewHeroApplication,
  updateArticle,
  updateIncident,
  updateMyAvatar,
  updateMyContact,
  updateNewsletterPreference,
  updateVillain,
} from "@/lib/api";
import { isModerationStatus } from "@/lib/format";
import {
  adminArticleRoute,
  adminIncidentRoute,
  adminVillainRoute,
  routes,
  villainSheetRoute,
} from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import {
  avatarErrorMessage,
  contactFailureState,
  readContactValues,
  validateAvatar,
  validateContact,
  type ContactField,
} from "@/lib/validation/account";
import {
  readArticleValues,
  readIncidentEditValues,
  readIncidentTypeValues,
  readVillainValues,
  toIncidentTypeInput,
  toArticleInput,
  toIncidentUpdate,
  toVillainInput,
  validateArticle,
  validateIncidentEdit,
  validateIncidentType,
  validateVillain,
  type ArticleField,
  type IncidentEditField,
  type IncidentTypeField,
  type VillainField,
} from "@/lib/validation/admin";
import { apiFailureMessage, hasFieldErrors, type FormState } from "@/lib/validation/form-state";
import { OUTCOME_NOTES_MAX_LENGTH, validateRejectionReason } from "@/lib/validation/hero";
import type {
  AccountModerationAction,
  AccountModerationResult,
  EmailChangeReviewResult,
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

/**
 * Arbitrage d'un désaccord : un membre de l'équipe a refusé le résultat proposé, la modération
 * tranche et clôt l'intervention.
 */
export async function arbitrateInterventionAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const incidentId = String(formData.get("id") ?? "");
  const interventionId = String(formData.get("interventionId") ?? "");
  const outcome = String(formData.get("outcome") ?? "");
  const notes = String(formData.get("notes") ?? "").trim();
  if (
    !incidentId ||
    !interventionId ||
    (outcome !== "reussi" && outcome !== "echoue") ||
    notes.length > OUTCOME_NOTES_MAX_LENGTH
  ) {
    return invalidRequest;
  }

  const result = await arbitrateIntervention(token, { incidentId, interventionId }, outcome, notes);
  return settle(
    result,
    [routes.adminIncidents, adminIncidentRoute(incidentId), routes.heroIncidents],
    outcome === "reussi"
      ? "Intervention close comme réussie : l'incident est résolu."
      : "Intervention close en échec : l'incident est de nouveau proposé aux super-héros.",
  );
}

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

/* ---------- Super-vilains ---------- */

/** Création (sans `id`, puis redirection vers la fiche) ou modification d'un super-vilain. */
export async function saveVillainAction(
  _previousState: FormState<VillainField>,
  formData: FormData,
): Promise<FormState<VillainField>> {
  const token = await requireAdminSession();
  const id = String(formData.get("id") ?? "");
  const values = readVillainValues(formData);
  const types = await getIncidentTypes(token);
  const fieldErrors = validateVillain(values, { typeIds: types.map(({ id: typeId }) => typeId) });
  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const input = toVillainInput(values);
  const result = id ? await updateVillain(token, id, input) : await createVillain(token, input);
  if (!result.ok) {
    switch (result.error) {
      case "alias-already-used":
        return {
          status: "error",
          fieldErrors: { alias: "Un super-vilain porte déjà cet alias." },
          values,
        };
      case "not-found":
        return {
          status: "error",
          message: "Ce super-vilain n'existe plus.",
          fieldErrors: {},
          values,
        };
      case "invalid-fields":
        return {
          status: "error",
          message: "Certains champs n'ont pas été acceptés.",
          fieldErrors: {
            alias: result.fields.alias && "Cet alias n'a pas été accepté.",
            typeIds: result.fields.typeIds && "Une spécialité n'a pas été acceptée.",
            threatLevel: result.fields.threatLevel && "Ce niveau n'a pas été accepté.",
            status: result.fields.status && "Ce statut n'a pas été accepté.",
          },
          values,
        };
      default:
        return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
    }
  }

  // Encyclopédie et formulaires de déclaration (liste des vilains) affichent aussitôt la fiche.
  const { villain } = result;
  for (const path of [
    routes.adminVillains,
    adminVillainRoute(villain.id),
    routes.encyclopedia,
    villainSheetRoute(villain.id),
    routes.profileIncidents,
    routes.heroDeclare,
    routes.adminReportIncident,
  ]) {
    revalidatePath(path);
  }
  if (!id) redirect(adminVillainRoute(villain.id));
  return {
    status: "success",
    message: `Fiche de « ${villain.alias} » enregistrée.`,
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
  if (!result.ok) {
    return { status: "error", message: emailChangeReviewErrorMessage(result), fieldErrors: {} };
  }

  for (const path of [routes.admin, routes.adminAccounts]) revalidatePath(path);
  return {
    status: "success",
    message: approve ? "Nouvelle adresse email activée." : "Changement d'adresse email refusé.",
    fieldErrors: {},
  };
}

function emailChangeReviewErrorMessage(
  result: Extract<EmailChangeReviewResult, { ok: false }>,
): string {
  switch (result.error) {
    case "not-found":
      return "Cette demande n'existe plus : son titulaire l'a peut-être remplacée par une nouvelle.";
    case "already-reviewed":
      return "Cette demande a déjà été traitée.";
    case "not-confirmed":
      return "Le titulaire de la nouvelle adresse ne l'a pas encore confirmée : vous pouvez seulement refuser la demande.";
    case "email-already-used":
      return "Cette adresse est maintenant utilisée par un autre compte : refusez la demande.";
    default:
      return apiFailureMessage(result);
  }
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

  const film = parseHeroVideoKey(request.id);
  if (!film) return invalidRequest;

  const result = await moderateVideo(token, film.heroId, film.filmId, request.decision);
  return settle(result, [routes.adminVideos], decisionMessages[request.decision]);
}

export async function addHeroVideoAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const heroId = String(formData.get("heroId") ?? "");
  const imdbId = String(formData.get("imdbId") ?? "");
  if (!heroId || !/^tt\d{7,}$/.test(imdbId)) return invalidRequest;

  const result = await addHeroVideo(token, heroId, imdbId);
  return settle(
    result,
    [routes.adminVideos, routes.videos],
    "Film ajouté et publié dans la vidéothèque du super-héros.",
  );
}

export async function deleteVideoAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const film = parseHeroVideoKey(String(formData.get("id") ?? ""));
  if (!film) return invalidRequest;

  const result = await deleteVideo(token, film.heroId, film.filmId);
  return settle(result, [routes.adminVideos], "Film supprimé.");
}

/* ---------- Films des super-vilains ---------- */

export async function addVillainFilmAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const villainId = String(formData.get("villainId") ?? "");
  const imdbId = String(formData.get("imdbId") ?? "");
  if (!villainId || !/^tt\d{7,}$/.test(imdbId)) return invalidRequest;

  const result = await addVillainFilm(token, villainId, imdbId);
  return settle(
    result,
    [adminVillainRoute(villainId), villainSheetRoute(villainId)],
    "Film ajouté à la fiche du super-vilain.",
  );
}

/** Le formulaire de suppression envoie la clé `villainId/filmId` dans `id`. */
export async function deleteVillainFilmAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const [villainId, filmId, ...rest] = String(formData.get("id") ?? "").split("/");
  if (!villainId || !filmId || rest.length > 0) return invalidRequest;

  const result = await deleteVillainFilm(token, villainId, filmId);
  return settle(
    result,
    [adminVillainRoute(villainId), villainSheetRoute(villainId)],
    "Film retiré de la fiche.",
  );
}

/* ---------- Commentaires ---------- */

export async function moderateReviewAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const request = readDecision(formData);
  if (!request) return invalidRequest;

  const result = await moderateReview(token, request.id, request.decision);
  return settle(
    result,
    [routes.adminComments],
    {
      "en-attente": "Avis remis en attente.",
      valide: "Avis publié : il compte de nouveau dans la note du super-héros.",
      rejete: "Avis masqué : il ne compte plus dans la note du super-héros.",
    }[request.decision],
  );
}

export async function moderateReplyAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const request = readDecision(formData);
  if (!request) return invalidRequest;

  const result = await moderateReply(token, request.id, request.decision);
  return settle(
    result,
    [routes.adminComments],
    {
      "en-attente": "Réponse remise en attente.",
      valide: "Réponse publiée.",
      rejete: "Réponse masquée.",
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

/* ---------- Mon profil ---------- */

/*
 * Profil du modérateur connecté : mêmes routes `/me` que les autres rôles. Son adresse email ne se
 * change pas ici : personne ne pourrait valider sa demande (voir backend routes_account.go).
 */

export async function updateModeratorContactAction(
  _previousState: FormState<ContactField>,
  formData: FormData,
): Promise<FormState<ContactField>> {
  const token = await requireAdminSession();
  const values = readContactValues(formData);
  const fieldErrors = validateContact(values);

  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await updateMyContact(token, {
    mobilePhone: values.mobilePhone.replace(/[\s.-]/g, ""),
    address: {
      street: values.street,
      complement: values.complement || null,
      postalCode: values.postalCode,
      city: values.city,
      country: values.country,
    },
  });

  if (!result.ok) return contactFailureState(result, values);

  revalidatePath(routes.adminProfile);
  return { status: "success", message: "Vos coordonnées ont été mises à jour.", fieldErrors: {} };
}

export async function updateModeratorAvatarAction(
  _previousState: FormState<"avatar">,
  formData: FormData,
): Promise<FormState<"avatar">> {
  const token = await requireAdminSession();
  const file = formData.get("avatar");
  const avatar = file instanceof File ? file : null;
  const error = validateAvatar(avatar);

  if (error || !avatar) return { status: "error", fieldErrors: { avatar: error } };

  const result = await updateMyAvatar(token, avatar);
  if (!result.ok) return { status: "error", message: avatarErrorMessage(result), fieldErrors: {} };

  revalidatePath(routes.adminProfile);
  return {
    status: "success",
    message: "Votre photo de profil a été mise à jour.",
    fieldErrors: {},
  };
}

export async function updateModeratorNewsletterAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireAdminSession();
  const subscribed = formData.get("newsletter") === "on";

  const result = await updateNewsletterPreference(token, subscribed);
  if (!result.ok) return { status: "error", message: apiFailureMessage(result), fieldErrors: {} };

  revalidatePath(routes.adminProfile);
  return {
    status: "success",
    message: subscribed
      ? "Vous êtes abonné·e à la newsletter."
      : "Vous ne recevrez plus la newsletter.",
    fieldErrors: {},
  };
}
