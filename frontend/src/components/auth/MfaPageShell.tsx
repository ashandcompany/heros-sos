import type { ReactNode } from "react";
import { logoutAction } from "@/app/actions/auth";
import { Container } from "@/components/ui/Container";

/**
 * Cadre des pages de la double authentification après le mot de passe : la session est limitée, la seule
 * autre issue est de renoncer et de se déconnecter.
 */
export function MfaPageShell({
  title,
  intro,
  children,
}: {
  title: string;
  intro: string;
  children: ReactNode;
}) {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface p-6 shadow-[3px_3px_0_0_var(--color-shadow)] sm:p-8">
          <h1 className="font-display text-3xl font-bold">{title}</h1>
          <p className="mt-2 mb-8 text-muted">{intro}</p>
          {children}
        </div>
        <form action={logoutAction} className="mt-6 text-center">
          <button
            type="submit"
            className="rounded-sm text-sm font-medium text-accent hover:underline"
          >
            Annuler et me déconnecter
          </button>
        </form>
      </div>
    </Container>
  );
}
