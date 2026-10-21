"use server";

import { revalidatePath } from "next/cache";
import { routes } from "@/lib/routes";
import {
  requestEmailChange,
  requestHeroStatus,
  updateMyAvatar,
  updateMyContact,
  updateNewsletterPreference,
} from "@/lib/api";
import { requireCitizenSession } from "@/lib/server-session";
import {
  avatarErrorMessage,
  contactErrorState,
  emailChangeErrorMessage,
  readContactValues,
  validateAvatar,
  validateContact,
  validateEmail,
  type ContactField,
} from "@/lib/validation/account";
import { apiFailureMessage, hasFieldErrors, type FormState } from "@/lib/validation/form-state";
import {
  heroRequestFieldErrors,
  readHeroRequestValues,
  validateHeroRequest,
  type HeroRequestField,
} from "@/lib/validation/hero";

export async function updateContactAction(
  _previousState: FormState<ContactField>,
  formData: FormData,
): Promise<FormState<ContactField>> {
  const token = await requireCitizenSession();
  const values = readContactValues(formData);
  const fieldErrors = validateContact(values);

  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await updateMyContact(token, {
    mobilePhone: values.mobilePhone.replace(/[\s.-]/g, ""),
    address: {
      street: values.street,
      // Chaîne vide : efface le complément (`null` le conserverait).
      complement: values.complement,
      postalCode: values.postalCode,
      city: values.city,
      country: values.country,
    },
  });

  if (!result.ok) return contactErrorState(result, values);

  revalidatePath(routes.profile);
  return { status: "success", message: "Vos coordonnées ont été mises à jour.", fieldErrors: {} };
}

export async function updateAvatarAction(
  _previousState: FormState<"avatar">,
  formData: FormData,
): Promise<FormState<"avatar">> {
  const token = await requireCitizenSession();
  const file = formData.get("avatar");
  const avatar = file instanceof File ? file : null;
  const error = validateAvatar(avatar);

  if (error || !avatar) return { status: "error", fieldErrors: { avatar: error } };

  const result = await updateMyAvatar(token, avatar);
  if (!result.ok) return { status: "error", fieldErrors: { avatar: avatarErrorMessage(result) } };

  revalidatePath(routes.profile);
  return {
    status: "success",
    message: "Votre photo de profil a été mise à jour.",
    fieldErrors: {},
  };
}

export async function requestEmailChangeAction(
  _previousState: FormState<"email">,
  formData: FormData,
): Promise<FormState<"email">> {
  const token = await requireCitizenSession();
  const email = String(formData.get("email") ?? "").trim();
  const error = validateEmail(email);

  if (error) return { status: "error", fieldErrors: { email: error }, values: { email } };

  const result = await requestEmailChange(token, email);
  if (!result.ok) {
    return {
      status: "error",
      fieldErrors: { email: emailChangeErrorMessage(result) },
      values: { email },
    };
  }

  revalidatePath(routes.profile);
  return {
    status: "success",
    message:
      "Demande envoyée : la nouvelle adresse sera active après validation par la modération.",
    fieldErrors: {},
  };
}

export async function updateNewsletterAction(
  _previousState: FormState,
  formData: FormData,
): Promise<FormState> {
  const token = await requireCitizenSession();
  const subscribed = formData.get("newsletter") === "on";

  const result = await updateNewsletterPreference(token, subscribed);
  if (!result.ok) return { status: "error", message: apiFailureMessage(result), fieldErrors: {} };

  revalidatePath(routes.profile);
  return {
    status: "success",
    message: subscribed
      ? "Vous êtes abonné·e à la newsletter."
      : "Vous ne recevrez plus la newsletter.",
    fieldErrors: {},
  };
}

/* ---------- Demande de statut super-héros ---------- */

const heroRequestErrorMessages = {
  "already-pending": "Votre demande est déjà en cours d'examen.",
  "already-hero": "Vous êtes déjà super-héros : rechargez la page pour ouvrir votre espace.",
  "not-eligible": "Votre compte ne peut pas faire cette demande. Contactez le Bureau.",
} as const;

export async function requestHeroStatusAction(
  _previousState: FormState<HeroRequestField>,
  formData: FormData,
): Promise<FormState<HeroRequestField>> {
  const token = await requireCitizenSession();
  const values = readHeroRequestValues(formData);
  const fieldErrors = validateHeroRequest(values);

  if (hasFieldErrors(fieldErrors)) return { status: "error", fieldErrors, values };

  const result = await requestHeroStatus(token, values);
  if (!result.ok) {
    switch (result.error) {
      case "alias-already-used":
        return {
          status: "error",
          fieldErrors: { alias: "Ce pseudo est déjà utilisé : choisissez-en un autre." },
          values,
        };
      case "invalid-fields":
        return { status: "error", fieldErrors: heroRequestFieldErrors(result.fields), values };
      case "already-pending":
      case "already-hero":
      case "not-eligible":
        return {
          status: "error",
          message: heroRequestErrorMessages[result.error],
          fieldErrors: {},
          values,
        };
      default:
        return { status: "error", message: apiFailureMessage(result), fieldErrors: {}, values };
    }
  }

  revalidatePath(routes.profile);
  return {
    status: "success",
    message: "Demande envoyée : le Bureau l'examinera dans les meilleurs délais.",
    fieldErrors: {},
  };
}
