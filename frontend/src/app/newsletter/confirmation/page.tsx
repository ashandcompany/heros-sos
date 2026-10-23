import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { confirmNewsletterAction } from "@/app/actions/newsletter";
import { Container } from "@/components/ui/Container";
import { TokenConfirmationForm } from "@/components/ui/TokenConfirmationForm";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Confirmation de la newsletter",
  description: "Confirmez votre abonnement à la newsletter Heros-SOS avec le lien reçu par email.",
  // Le jeton est dans l'URL : ni indexation, ni transmission dans l'en-tête Referer.
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

/** Page du lien reçu par mail après la demande d'abonnement (`/newsletter/confirmation?token=…`). */
export default function NewsletterConfirmationPage({
  searchParams,
}: PageProps<"/newsletter/confirmation">) {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Newsletter</h1>
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
        renouvelez votre demande depuis le formulaire en bas de n&apos;importe quelle page du site.
      </p>
    );
  }

  return (
    <>
      <p className="mt-2 mb-8 text-muted">
        Dernière étape : confirmez votre abonnement pour recevoir la newsletter.
      </p>
      <TokenConfirmationForm
        token={token}
        action={confirmNewsletterAction}
        submitLabel="Confirmer mon abonnement"
        pendingLabel="Confirmation…"
        success={{
          title: "Abonnement confirmé.",
          message: "Merci ! Vous recevrez désormais la newsletter du Bureau.",
        }}
        invalid={{
          title: "Lien invalide ou expiré.",
          message:
            "Ce lien a expiré, ou l'abonnement a été retiré entre-temps. Renouvelez votre demande depuis le formulaire en bas de n'importe quelle page du site.",
        }}
        unavailable="La confirmation est momentanément indisponible. Réessayez dans quelques instants."
        continueHref={routes.home}
        continueLabel="Retour à l'accueil"
      />
    </>
  );
}
