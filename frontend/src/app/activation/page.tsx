import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { ActivationForm, ResendActivationForm } from "@/components/auth/ActivationForm";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Activation du compte",
  description: "Activez votre compte Heros-SOS avec le lien reçu par email.",
  // Le jeton est dans l'URL : ni indexation, ni transmission dans l'en-tête Referer.
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

/** Page du lien reçu par mail après l'inscription (`/activation?token=…`). */
export default function ActivationPage({ searchParams }: PageProps<"/activation">) {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Activation du compte</h1>
          <Suspense>
            <ActivationContent searchParams={searchParams} />
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

async function ActivationContent({
  searchParams,
}: {
  searchParams: PageProps<"/activation">["searchParams"];
}) {
  const { token } = await searchParams;

  if (typeof token !== "string" || !token) {
    return (
      <>
        <p className="mt-2 mb-8 text-muted">
          Ce lien d&apos;activation est incomplet. Copiez le lien entier depuis le mail reçu, ou
          demandez-en un nouveau.
        </p>
        <ResendActivationForm />
      </>
    );
  }

  return (
    <>
      <p className="mt-2 mb-8 text-muted">
        Dernière étape : confirmez l&apos;activation pour pouvoir vous connecter.
      </p>
      <ActivationForm token={token} />
    </>
  );
}
