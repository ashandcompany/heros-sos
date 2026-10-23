import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { unsubscribeNewsletterAction } from "@/app/actions/newsletter";
import { Container } from "@/components/ui/Container";
import { TokenConfirmationForm } from "@/components/ui/TokenConfirmationForm";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Désinscription de la newsletter",
  description: "Ne plus recevoir la newsletter Heros-SOS.",
  // Le jeton est dans l'URL : ni indexation, ni transmission dans l'en-tête Referer.
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

/** Page du lien de désinscription que porte chaque newsletter (`/newsletter/desinscription?token=…`). */
export default function NewsletterUnsubscribePage({
  searchParams,
}: PageProps<"/newsletter/desinscription">) {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Se désabonner</h1>
          <Suspense>
            <UnsubscribeContent searchParams={searchParams} />
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

async function UnsubscribeContent({
  searchParams,
}: {
  searchParams: PageProps<"/newsletter/desinscription">["searchParams"];
}) {
  const { token } = await searchParams;

  if (typeof token !== "string" || !token) {
    return (
      <p className="mt-2 text-muted">
        Ce lien de désinscription est incomplet. Copiez le lien entier depuis la newsletter reçue.
        Vous pouvez aussi nous écrire depuis la page de contact.
      </p>
    );
  }

  return (
    <>
      <p className="mt-2 mb-8 text-muted">
        Confirmez pour ne plus recevoir la newsletter. Vous pourrez vous réabonner à tout moment
        depuis le pied de page du site.
      </p>
      <TokenConfirmationForm
        token={token}
        action={unsubscribeNewsletterAction}
        submitLabel="Me désabonner"
        pendingLabel="Désinscription…"
        success={{
          title: "Vous êtes désabonné·e.",
          message: "Vous ne recevrez plus la newsletter. Merci de nous avoir lus !",
        }}
        invalid={{
          title: "Lien invalide.",
          message:
            "Ce lien de désinscription est incomplet ou n'est pas valable. Copiez le lien entier depuis la newsletter, ou écrivez-nous depuis la page de contact.",
        }}
        unavailable="La désinscription est momentanément indisponible. Réessayez dans quelques instants."
        continueHref={routes.home}
        continueLabel="Retour à l'accueil"
      />
    </>
  );
}
