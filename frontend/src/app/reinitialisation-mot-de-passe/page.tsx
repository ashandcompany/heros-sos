import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { InvalidResetLink, ResetPasswordForm } from "@/components/auth/ResetPasswordForm";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Nouveau mot de passe",
  description: "Choisissez un nouveau mot de passe avec le lien reçu par email.",
  // Le jeton est dans l'URL : ni indexation, ni transmission dans l'en-tête Referer.
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

/** Page du lien reçu par mail après une demande de mot de passe oublié (`?token=…`). */
export default function PasswordResetPage({
  searchParams,
}: PageProps<"/reinitialisation-mot-de-passe">) {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Nouveau mot de passe</h1>
          <Suspense>
            <PasswordResetContent searchParams={searchParams} />
          </Suspense>
        </div>

        <p className="mt-6 text-center text-sm">
          <Link href={routes.login} className="rounded-sm font-medium text-accent hover:underline">
            <span aria-hidden="true">←</span> Retour à la connexion
          </Link>
        </p>
      </div>
    </Container>
  );
}

async function PasswordResetContent({
  searchParams,
}: {
  searchParams: PageProps<"/reinitialisation-mot-de-passe">["searchParams"];
}) {
  const { token } = await searchParams;

  if (typeof token !== "string" || !token) {
    return (
      <div className="mt-6">
        <InvalidResetLink />
      </div>
    );
  }

  return (
    <>
      <p className="mt-2 mb-8 text-muted">
        Choisissez le nouveau mot de passe de votre compte. Vous serez ensuite déconnecté de tous
        vos appareils.
      </p>
      <ResetPasswordForm token={token} />
    </>
  );
}
