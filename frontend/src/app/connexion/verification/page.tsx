import type { Metadata } from "next";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { MfaPageShell } from "@/components/auth/MfaPageShell";
import { MfaVerificationForm } from "@/components/auth/MfaVerificationForm";
import { getMfaSession } from "@/lib/api";
import { SESSION_COOKIE } from "@/lib/cookies";
import { accountHomeByRole, routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Double authentification",
  robots: { index: false, follow: false },
};

/** Seconde étape de la connexion d'un compte protégé par un code (TOTP) : la session est limitée. */
export default function MfaVerificationPage() {
  return (
    <Suspense>
      <VerificationContent />
    </Suspense>
  );
}

async function VerificationContent() {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  const session = token ? await getMfaSession(token) : null;

  if (!session) redirect(routes.login);
  // Rien à valider ici : activation à faire d'abord, ou session déjà complète.
  if (session.status === "enrollment-required") redirect(routes.loginTwoFactorSetup);
  if (session.status !== "verification-required") redirect(accountHomeByRole[session.user.role]);

  return (
    <MfaPageShell
      title="Double authentification"
      intro="Saisissez le code à 6 chiffres affiché par votre application d'authentification pour terminer la connexion."
    >
      <MfaVerificationForm />
    </MfaPageShell>
  );
}
