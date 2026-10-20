import type { Metadata } from "next";
import Link from "next/link";
import { ForgotPasswordForm } from "@/components/auth/ForgotPasswordForm";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Mot de passe oublié",
  description: "Recevez un lien par email pour réinitialiser le mot de passe de votre compte.",
};

/** Accessible uniquement aux visiteurs non connectés (redirection gérée par src/proxy.ts). */
export default function ForgotPasswordPage() {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Mot de passe oublié</h1>
          <p className="mt-2 mb-8 text-muted">
            Saisissez l&apos;adresse email de votre compte : nous vous enverrons un lien pour
            choisir un nouveau mot de passe.
          </p>
          <ForgotPasswordForm />
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
