"use server";

import { revalidatePath } from "next/cache";
import { routes } from "@/lib/routes";
import { declareIncident, getIncidentTypes, getVillains, rateIntervention } from "@/lib/api";
import {
  requireAdminSession,
  requireCitizenSession,
  requireRole,
  requireHeroSession,
} from "@/lib/server-session";
import { isUuid } from "@/lib/offline/commands";
import { hasFieldErrors, type FormState } from "@/lib/validation/form-state";
import {
  readDeclarationValues,
  toDeclaration,
  validateDeclaration,
  type DeclarationField,
} from "@/lib/validation/incident";

export type DeclareIncidentFormAction = (
  previousState: FormState<DeclarationField>,
  formData: FormData,
) => Promise<FormState<DeclarationField>>;

export async function declareIncidentAction(
  _previousState: FormState<DeclarationField>,
  formData: FormData,
): Promise<FormState<DeclarationField>> {
  return submitDeclaration(await requireCitizenSession(), formData);
}

/** Signalement par un modérateur, depuis le back-office (validé d'office par l'API). */
export async function adminDeclareIncidentAction(
  _previousState: FormState<DeclarationField>,
  formData: FormData,
): Promise<FormState<DeclarationField>> {
  const state = await submitDeclaration(await requireAdminSession(), formData);
  if (state.status === "success") {
    revalidatePath(routes.admin);
    revalidatePath(routes.adminIncidents);
  }
  return state;
}

async function submitDeclaration(
  token: string,
  formData: FormData,
): Promise<FormState<DeclarationField>> {
  const values = readDeclarationValues(formData);
  // Date convertie en ISO par le navigateur (fuseau de l'utilisateur) quand JavaScript est actif.
  const occurredAtIso = String(formData.get("occurredAtIso") ?? "");
  const validated = { ...values, occurredAt: occurredAtIso || values.occurredAt };

  const [villains, types] = await Promise.all([getVillains(), getIncidentTypes(token)]);
  const fieldErrors = validateDeclaration(validated, {
    villainIds: villains.map(({ id }) => id),
    typeIds: types.map(({ id }) => id),
  });

  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  // Identifiant choisi par l'appareil (champ caché) : si la réponse se perd et que l'envoi est rejoué hors ligne,
  // l'API reconnaît l'incident au lieu d'en créer un second.
  const incidentId = String(formData.get("incidentId") ?? "");
  const declaration = toDeclaration(validated);
  const result = await declareIncident(
    token,
    isUuid(incidentId) ? { ...declaration, id: incidentId } : declaration,
  );
  if (!result.ok) return { status: "error", message: result.error, fieldErrors: {}, values };

  revalidatePath(routes.profileIncidents);
  return {
    status: "success",
    message: "Incident déclaré. Les super-héros à proximité ont été prévenus.",
    fieldErrors: {},
  };
}

type RatingField = "rating" | "comment";

const COMMENT_MAX_LENGTH = 500;

export async function rateInterventionAction(
  _previousState: FormState<RatingField>,
  formData: FormData,
): Promise<FormState<RatingField>> {
  // Le déclarant note les héros intervenus : un citoyen, ou un héros / modérateur qui a signalé l'incident.
  const { token } = await requireRole(["citoyen", "super-heros", "admin"]);
  const interventionId = String(formData.get("interventionId") ?? "");
  const heroId = String(formData.get("heroId") ?? "");
  const rating = Number(formData.get("rating"));
  const comment = String(formData.get("comment") ?? "").trim();

  const fieldErrors = {
    rating:
      Number.isInteger(rating) && rating >= 1 && rating <= 5
        ? undefined
        : "Choisissez une note de 1 à 5.",
    comment:
      comment.length > COMMENT_MAX_LENGTH
        ? `Le commentaire ne doit pas dépasser ${COMMENT_MAX_LENGTH} caractères.`
        : undefined,
  };

  if (!interventionId || !heroId) {
    return { status: "error", message: "Requête invalide.", fieldErrors: {}, values: { comment } };
  }
  if (hasFieldErrors(fieldErrors)) {
    return { status: "error", fieldErrors, values: { rating: String(rating || ""), comment } };
  }

  const result = await rateIntervention(token, interventionId, heroId, {
    rating,
    comment: comment || null,
  });
  if (!result.ok) {
    return { status: "error", message: result.error, fieldErrors: {}, values: { comment } };
  }

  revalidatePath(routes.profileInterventions);
  revalidatePath(routes.heroRate);
  revalidatePath(routes.adminRate);
  return { status: "success", message: "Merci, votre avis a été publié.", fieldErrors: {} };
}

/** Signalement par un super-héros, comme un citoyen : il attend la vérification de la modération. */
export async function heroDeclareIncidentAction(
  _previousState: FormState<DeclarationField>,
  formData: FormData,
): Promise<FormState<DeclarationField>> {
  const state = await submitDeclaration((await requireHeroSession()).token, formData);
  if (state.status === "success") {
    revalidatePath(routes.heroDeclare);
    revalidatePath(routes.heroIncidents);
  }
  return state;
}
