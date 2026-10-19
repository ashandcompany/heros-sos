"use server";

import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { requestPasswordReset } from "@/lib/api";
import { SESSION_COOKIE } from "@/lib/cookies";
import { getAccountHome } from "@/lib/session";

export type ForgotPasswordFormState =
  { status: "idle" } | { status: "success" } | { status: "error"; message: string; email: string };

const EMAIL_PATTERN = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;

export async function forgotPasswordAction(
  _previousState: ForgotPasswordFormState,
  formData: FormData,
): Promise<ForgotPasswordFormState> {
  // Le proxy protège déjà la page, mais une Server Action reste appelable directement en POST.
  const accountHome = await getAccountHome((await cookies()).get(SESSION_COOKIE)?.value);
  if (accountHome) redirect(accountHome);

  const email = String(formData.get("email") ?? "").trim();

  if (!EMAIL_PATTERN.test(email)) {
    return { status: "error", message: "Veuillez saisir une adresse email valide.", email };
  }

  const result = await requestPasswordReset(email);

  if (!result.ok) {
    return {
      status: "error",
      message: "La demande n'a pas pu aboutir. Veuillez réessayer plus tard.",
      email,
    };
  }

  return { status: "success" };
}
