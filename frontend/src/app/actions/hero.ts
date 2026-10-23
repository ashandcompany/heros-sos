"use server";

import { revalidatePath } from "next/cache";
import {
  addMyVideo,
  deleteHeroLocation,
  deleteMyVideo,
  finishIntervention,
  grantHeroLocationConsent,
  inviteHeroes,
  joinIntervention,
  replyToReview,
  requestHeroEmailChange,
  respondToInvitation,
  revokeHeroLocationConsent,
  saveHeroLocation,
  setIncidentPriority,
  takeChargeOfIncident,
  updateMyHeroAvatar,
  updateMyHeroContact,
  voteOnOutcome,
} from "@/lib/api";
import { isIncidentPriority } from "@/lib/format";
import { heroIncidentRoute, routes } from "@/lib/routes";
import { requireHeroSession } from "@/lib/server-session";
import {
  avatarErrorMessage,
  contactFailureState,
  emailChangeFailureState,
  readContactValues,
  validateAvatar,
  validateContact,
  validateEmail,
  type ContactField,
} from "@/lib/validation/account";
import { hasFieldErrors, type FormState } from "@/lib/validation/form-state";
import { OUTCOME_NOTES_MAX_LENGTH, REPLY_MAX_LENGTH } from "@/lib/validation/hero";
import {
  readFreeVideoValues,
  validateFreeVideo,
  type FreeVideoValues,
} from "@/lib/validation/video";
import type {
  FilmSource,
  HeroPosition,
  InterventionOutcome,
  InterventionRef,
  MutationResult,
} from "@/types";

/*
 * Server Actions de l'espace super-héros. Chacune revérifie la session (le proxy ne suffit pas)
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
  for (const path of paths) revalidatePath(path);
  return { status: "success", message, fieldErrors: {} };
}

/** Pages qui affichent l'état d'un incident. */
const incidentPaths = (id: string) => [
  routes.heroDashboard,
  routes.heroIncidents,
  heroIncidentRoute(id),
  routes.home,
  routes.profileIncidents,
  routes.adminIncidents,
];

/* ---------- Incidents ---------- */

export async function setPriorityAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const id = String(formData.get("id") ?? "");
  const priority = String(formData.get("priority") ?? "");
  if (!id || !isIncidentPriority(priority)) return invalidRequest;

  const result = await setIncidentPriority(token, id, priority);
  return settle(result, incidentPaths(id), "Priorité mise à jour.");
}

export async function takeChargeAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const id = String(formData.get("id") ?? "");
  if (!id) return invalidRequest;

  const result = await takeChargeOfIncident(token, id);
  return settle(
    result,
    incidentPaths(id),
    "Vous avez pris en charge l'incident. Le citoyen est informé de votre intervention.",
  );
}

const outcomeMessages: Record<InterventionOutcome, string> = {
  reussi: "Intervention réussie : l'incident est résolu. Bravo !",
  echoue: "Intervention terminée en échec : l'incident est de nouveau proposé aux super-héros.",
};

/** Intervention visée par un formulaire de l'équipe (champs cachés `id` et `interventionId`). */
function readInterventionRef(formData: FormData): InterventionRef | null {
  const incidentId = String(formData.get("id") ?? "");
  const interventionId = String(formData.get("interventionId") ?? "");
  return incidentId && interventionId ? { incidentId, interventionId } : null;
}

const readOutcome = (formData: FormData): InterventionOutcome | null => {
  const outcome = String(formData.get("outcome") ?? "");
  return outcome === "reussi" || outcome === "echoue" ? outcome : null;
};

export async function finishInterventionAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const ref = readInterventionRef(formData);
  const outcome = readOutcome(formData);
  const notes = String(formData.get("notes") ?? "").trim();
  if (!ref || !outcome || notes.length > OUTCOME_NOTES_MAX_LENGTH) return invalidRequest;

  const result = await finishIntervention(token, ref, outcome, notes);
  if (!result.ok) return { status: "error", message: result.error, fieldErrors: {} };
  return settle(
    result,
    [...incidentPaths(ref.incidentId), routes.profileInterventions],
    result.closed
      ? outcomeMessages[outcome]
      : "Résultat soumis à l'équipe : l'intervention sera close quand tous ses membres l'auront validé.",
  );
}

export async function joinInterventionAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const ref = readInterventionRef(formData);
  if (!ref) return invalidRequest;

  const result = await joinIntervention(token, ref);
  return settle(result, incidentPaths(ref.incidentId), "Vous avez rejoint l'équipe.");
}

export async function inviteHeroesAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const ref = readInterventionRef(formData);
  const heroIds = [...new Set(formData.getAll("heroIds").map(String).filter(Boolean))];
  if (!ref) return invalidRequest;
  if (heroIds.length === 0) {
    return { status: "error", message: "Choisissez au moins un super-héros.", fieldErrors: {} };
  }

  const result = await inviteHeroes(token, ref, heroIds);
  return settle(
    result,
    [heroIncidentRoute(ref.incidentId)],
    heroIds.length > 1
      ? "Invitations envoyées : chaque super-héros doit confirmer sa présence."
      : "Invitation envoyée : le super-héros doit confirmer sa présence.",
  );
}

export async function respondToInvitationAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const ref = readInterventionRef(formData);
  const answer = String(formData.get("answer") ?? "");
  if (!ref || (answer !== "accept" && answer !== "decline")) return invalidRequest;

  const accept = answer === "accept";
  const result = await respondToInvitation(token, ref, accept);
  return settle(
    result,
    incidentPaths(ref.incidentId),
    accept ? "Vous faites partie de l'équipe." : "Invitation refusée.",
  );
}

export async function voteOnOutcomeAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const ref = readInterventionRef(formData);
  const vote = String(formData.get("vote") ?? "");
  if (!ref || (vote !== "approve" && vote !== "reject")) return invalidRequest;

  const approved = vote === "approve";
  const result = await voteOnOutcome(token, ref, approved);
  return settle(
    result,
    [...incidentPaths(ref.incidentId), routes.profileInterventions],
    approved
      ? "Vote enregistré. L'intervention sera close quand toute l'équipe aura validé."
      : "Résultat refusé : la modération va arbitrer.",
  );
}

/* ---------- Profil ---------- */

export async function updateHeroContactAction(
  _previousState: FormState<ContactField>,
  formData: FormData,
): Promise<FormState<ContactField>> {
  const { token } = await requireHeroSession();
  // Le profil d'un super-héros n'a pas de complément d'adresse.
  const values = { ...readContactValues(formData), complement: "" };
  const fieldErrors = validateContact(values);

  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await updateMyHeroContact(token, {
    mobilePhone: values.mobilePhone.replace(/[\s.-]/g, ""),
    address: {
      street: values.street,
      postalCode: values.postalCode,
      city: values.city,
      country: values.country,
    },
  });
  if (!result.ok) return contactFailureState(result, values);

  revalidatePath(routes.heroProfile);
  revalidatePath(routes.adminHeroes);
  return { status: "success", message: "Vos coordonnées ont été mises à jour.", fieldErrors: {} };
}

export async function updateHeroAvatarAction(
  _previousState: FormState<"avatar">,
  formData: FormData,
): Promise<FormState<"avatar">> {
  const { token } = await requireHeroSession();
  const file = formData.get("avatar");
  const avatar = file instanceof File ? file : null;
  const error = validateAvatar(avatar);

  if (error || !avatar) return { status: "error", fieldErrors: { avatar: error } };

  const result = await updateMyHeroAvatar(token, avatar);
  if (!result.ok) return { status: "error", message: avatarErrorMessage(result), fieldErrors: {} };

  for (const path of [routes.heroProfile, routes.home, routes.encyclopedia]) revalidatePath(path);
  return { status: "success", message: "Votre photo a été mise à jour.", fieldErrors: {} };
}

export async function requestHeroEmailChangeAction(
  _previousState: FormState<"email">,
  formData: FormData,
): Promise<FormState<"email">> {
  const { token } = await requireHeroSession();
  const email = String(formData.get("email") ?? "").trim();
  const error = validateEmail(email);

  if (error) return { status: "error", fieldErrors: { email: error }, values: { email } };

  const result = await requestHeroEmailChange(token, email);
  if (!result.ok) return emailChangeFailureState(result, email);

  revalidatePath(routes.heroProfile);
  return {
    status: "success",
    message:
      "Demande envoyée : un mail de confirmation vient d'être envoyé à la nouvelle adresse. Elle sera active une fois confirmée, puis validée par la modération.",
    fieldErrors: {},
  };
}

/* ---------- Vidéothèque ---------- */

export async function addVideoAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  let source: FilmSource;
  let values: FreeVideoValues | undefined;
  if (formData.has("imdbId")) {
    source = String(formData.get("imdbId"));
    if (!/^tt\d{7,}$/.test(source)) return invalidRequest;
  } else {
    // Vidéo libre, hors IMDb.
    source = values = readFreeVideoValues(formData);
    const fieldErrors = validateFreeVideo(values);
    if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };
  }

  const result = await addMyVideo(token, source);
  if (!result.ok && values) {
    return { status: "error", message: result.error, fieldErrors: {}, values };
  }
  return settle(
    result,
    [routes.heroVideos, routes.heroDashboard, routes.admin, routes.adminVideos],
    "Film ajouté : il sera visible des citoyens après validation par la modération.",
  );
}

export async function deleteMyVideoAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const id = String(formData.get("id") ?? "");
  if (!id) return invalidRequest;

  const result = await deleteMyVideo(token, id);
  return settle(
    result,
    [routes.heroVideos, routes.heroDashboard, routes.admin, routes.adminVideos],
    "Film retiré de votre vidéothèque.",
  );
}

/* ---------- Partage de position ---------- */

export async function setLocationSharingAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const enabled = formData.get("enabled");
  if (enabled !== "true" && enabled !== "false") return invalidRequest;

  const sharing = enabled === "true";
  const result = sharing
    ? await grantHeroLocationConsent(token)
    : await revokeHeroLocationConsent(token);
  // Le suivi de la position est rendu par le layout de l'espace héros : il doit relire le consentement.
  if (result.ok) revalidatePath(routes.heroSpace, "layout");
  return settle(
    result,
    [routes.heroProfile],
    sharing
      ? "Partage de position activé."
      : "Partage de position désactivé : votre position a été supprimée.",
  );
}

/**
 * Envoie la position mesurée par le navigateur. Appelée directement par le suivi de l'espace héros
 * (pas de formulaire) : les valeurs viennent du client et sont revérifiées ici.
 */
export async function saveHeroLocationAction(position: HeroPosition): Promise<MutationResult> {
  const { token } = await requireHeroSession();
  const { latitude, longitude, accuracyM } = position ?? {};
  const isCoordinate = (value: unknown, limit: number): value is number =>
    typeof value === "number" && Number.isFinite(value) && Math.abs(value) <= limit;
  if (!isCoordinate(latitude, 90) || !isCoordinate(longitude, 180)) {
    return { ok: false, error: "Position invalide." };
  }
  const accuracy =
    typeof accuracyM === "number" && Number.isFinite(accuracyM) && accuracyM >= 0
      ? Math.round(accuracyM)
      : undefined;
  return saveHeroLocation(token, { latitude, longitude, accuracyM: accuracy });
}

/** Supprime la position en gardant le consentement (« Arrêter pour l'instant »). */
export async function pauseHeroLocationAction(): Promise<MutationResult> {
  const { token } = await requireHeroSession();
  return deleteHeroLocation(token);
}

/* ---------- Réputation ---------- */

export async function replyToReviewAction(
  _previousState: FormState<"reply">,
  formData: FormData,
): Promise<FormState<"reply">> {
  const { token } = await requireHeroSession();
  const reviewId = String(formData.get("reviewId") ?? "");
  if (!reviewId) return { status: "error", message: "Requête invalide.", fieldErrors: {} };
  const reply = String(formData.get("reply") ?? "").trim();

  const error = !reply
    ? "Rédigez votre réponse."
    : reply.length > REPLY_MAX_LENGTH
      ? `La réponse ne doit pas dépasser ${REPLY_MAX_LENGTH} caractères.`
      : undefined;
  if (error) return { status: "error", fieldErrors: { reply: error }, values: { reply } };

  // Modification : un texte inchangé remettrait la réponse en vérification pour rien.
  const currentReply = formData.get("currentReply");
  const editing = currentReply !== null;
  if (editing && String(currentReply).trim() === reply) {
    return {
      status: "error",
      fieldErrors: { reply: "Votre réponse n'a pas changé." },
      values: { reply },
    };
  }

  const result = await replyToReview(token, reviewId, reply);
  if (!result.ok) {
    return { status: "error", message: result.error, fieldErrors: {}, values: { reply } };
  }

  revalidatePath(routes.heroReputation);
  return {
    status: "success",
    message: editing
      ? "Réponse modifiée : elle sera de nouveau publiée après vérification par la modération."
      : "Votre réponse a été publiée.",
    fieldErrors: {},
  };
}
