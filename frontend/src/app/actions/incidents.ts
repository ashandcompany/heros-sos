"use server";

import { revalidatePath } from "next/cache";
import { routes } from "@/lib/routes";
import { declareIncident, getVillains, rateIntervention } from "@/lib/api";
import { requireCitizenSession } from "@/lib/server-session";
import { hasFieldErrors, type FormState } from "@/lib/validation/form-state";
import {
  readDeclarationValues,
  toDeclaration,
  validateDeclaration,
  type DeclarationField,
} from "@/lib/validation/incident";

export async function declareIncidentAction(
  _previousState: FormState<DeclarationField>,
  formData: FormData,
): Promise<FormState<DeclarationField>> {
  const token = await requireCitizenSession();
  const values = readDeclarationValues(formData);
  // Date convertie en ISO par le navigateur (fuseau de l'utilisateur) quand JavaScript est actif.
  const occurredAtIso = String(formData.get("occurredAtIso") ?? "");
  const validated = { ...values, occurredAt: occurredAtIso || values.occurredAt };

  const villainIds = (await getVillains()).map(({ id }) => id);
  const fieldErrors = validateDeclaration(validated, villainIds);

  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await declareIncident(token, toDeclaration(validated));
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
  const token = await requireCitizenSession();
  const incidentId = String(formData.get("incidentId") ?? "");
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

  if (hasFieldErrors(fieldErrors)) {
    return { status: "error", fieldErrors, values: { rating: String(rating || ""), comment } };
  }

  const result = await rateIntervention(token, incidentId, { rating, comment: comment || null });
  if (!result.ok) {
    return { status: "error", message: result.error, fieldErrors: {}, values: { comment } };
  }

  revalidatePath(routes.profileInterventions);
  return { status: "success", message: "Merci, votre avis a été enregistré.", fieldErrors: {} };
}
