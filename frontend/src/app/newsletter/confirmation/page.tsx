import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { NewsletterConfirmationForm } from "@/components/layout/NewsletterConfirmationForm";
import { Container } from "@/components/ui/Container";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Confirmation de l'abonnement",
  description: "Confirmez votre abonnement à la newsletter Heros-SOS avec le lien reçu par email.",
  // Le jeton est dans l'URL : ni indexation, ni transmission dans l'en-tête Referer.
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

/** Page du lien reçu par mail après l'inscription à la newsletter (`/newsletter/confirmation?token=…`). */
export default function NewsletterConfirmationPage({
  searchParams,
}: PageProps<"/newsletter/confirmation">) {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Abonnement à la newsletter</h1>
          <Suspense>
            <ConfirmationContent searchParams={searchParams} />
          </Suspense>
        </div>

        <p className="mt-6 text-center text-sm">
          <Link href={routes.home} className="rounded-sm font-medium text-accent hover:underline">
            <span aria-hidden="true">←</span> Retour à l&apos;accueil
          </Link>
        </p>
      </div>
    </Container>
  );
}

async function ConfirmationContent({
  searchParams,
}: {
  searchParams: PageProps<"/newsletter/confirmation">["searchParams"];
}) {
  const { token } = await searchParams;

  if (typeof token !== "string" || !token) {
    return (
      <p className="mt-2 text-muted">
        Ce lien de confirmation est incomplet. Copiez le lien entier depuis le mail reçu, ou
        inscrivez-vous de nouveau depuis le bas de n&apos;importe quelle page.
      </p>
    );
  }

  return (
    <>
      <p className="mt-2 mb-8 text-muted">
        Dernière étape : confirmez que vous souhaitez recevoir le bulletin du Bureau.
      </p>
      <NewsletterConfirmationForm token={token} />
    </>
  );
}
