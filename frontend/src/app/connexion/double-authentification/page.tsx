import type { Metadata } from "next";
import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { Suspense } from "react";
import { MfaPageShell } from "@/components/auth/MfaPageShell";
import { MfaSetup } from "@/components/auth/MfaSetup";
import { getMfaSession } from "@/lib/api";
import { SESSION_COOKIE } from "@/lib/cookies";
import { accountHomeByRole, routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Activer la double authentification",
  robots: { index: false, follow: false },
};

/**
 * Activation obligatoire de la double authentification : un compte de modérateur donne accès à toutes les
 * données de la plateforme, sa session reste limitée tant qu'elle n'est pas activée.
 */
export default function MfaEnrollmentPage() {
  return (
    <Suspense>
      <EnrollmentContent />
    </Suspense>
  );
}

async function EnrollmentContent() {
  const token = (await cookies()).get(SESSION_COOKIE)?.value;
  const session = token ? await getMfaSession(token) : null;

  if (!session) redirect(routes.login);
  if (session.status === "verification-required") redirect(routes.loginVerification);
  if (session.status !== "enrollment-required") redirect(accountHomeByRole[session.user.role]);

  return (
    <MfaPageShell
      title="Activez la double authentification"
      intro="Votre compte de modérateur donne accès à toutes les données de la plateforme : il doit être protégé par un code en plus du mot de passe."
    >
      <MfaSetup
        continueHref={accountHomeByRole[session.user.role]}
        continueLabel="Accéder au back-office"
      />
    </MfaPageShell>
  );
}
