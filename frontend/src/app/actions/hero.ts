"use server";

import { revalidatePath } from "next/cache";
import {
  addMyVideo,
  deleteMyVideo,
  finishIntervention,
  replyToReview,
  requestHeroEmailChange,
  setIncidentPriority,
  takeChargeOfIncident,
  updateMyHeroAvatar,
  updateMyHeroContact,
} from "@/lib/api";
import { isIncidentPriority } from "@/lib/format";
import { heroIncidentRoute, routes } from "@/lib/routes";
import { requireHeroSession } from "@/lib/server-session";
import {
  readContactValues,
  validateAvatar,
  validateContact,
  validateEmail,
  type ContactField,
} from "@/lib/validation/account";
import { hasFieldErrors, type FormState } from "@/lib/validation/form-state";
import { REPLY_MAX_LENGTH } from "@/lib/validation/hero";
import type { InterventionOutcome, MutationResult } from "@/types";

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

export async function finishInterventionAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const id = String(formData.get("id") ?? "");
  const outcome = String(formData.get("outcome") ?? "");
  if (!id || (outcome !== "reussi" && outcome !== "echoue")) return invalidRequest;

  const result = await finishIntervention(token, id, outcome);
  return settle(
    result,
    [...incidentPaths(id), routes.profileInterventions],
    outcomeMessages[outcome],
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
  if (!result.ok) return { status: "error", message: result.error, fieldErrors: {}, values };

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
  if (!result.ok) return { status: "error", message: result.error, fieldErrors: {} };

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
  if (!result.ok) {
    return { status: "error", fieldErrors: { email: result.error }, values: { email } };
  }

  revalidatePath(routes.heroProfile);
  return {
    status: "success",
    message:
      "Demande envoyée : la nouvelle adresse sera active après validation par la modération.",
    fieldErrors: {},
  };
}

/* ---------- Vidéothèque ---------- */

export async function addVideoAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const { token } = await requireHeroSession();
  const imdbId = String(formData.get("imdbId") ?? "");
  if (!/^tt\d{7,}$/.test(imdbId)) return invalidRequest;

  const result = await addMyVideo(token, imdbId);
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

  const result = await replyToReview(token, reviewId, reply);
  if (!result.ok) {
    return { status: "error", message: result.error, fieldErrors: {}, values: { reply } };
  }

  revalidatePath(routes.heroReputation);
  return { status: "success", message: "Votre réponse a été publiée.", fieldErrors: {} };
}
