import type { Metadata } from "next";
import Link from "next/link";
import { Suspense } from "react";
import { confirmEmailChangeAction } from "@/app/actions/email-change";
import { Container } from "@/components/ui/Container";
import { TokenConfirmationForm } from "@/components/ui/TokenConfirmationForm";
import { routes } from "@/lib/routes";

export const metadata: Metadata = {
  title: "Confirmation de la nouvelle adresse email",
  description: "Confirmez la nouvelle adresse email de votre compte Heros-SOS.",
  // Le jeton est dans l'URL : ni indexation, ni transmission dans l'en-tête Referer.
  robots: { index: false, follow: false },
  referrer: "no-referrer",
};

/**
 * Page du lien reçu à la NOUVELLE adresse d'un compte (`/changement-email/confirmation?token=…`). Elle ne
 * demande pas de connexion : le lien s'ouvre souvent sur un autre appareil que celui de la demande.
 */
export default function EmailChangeConfirmationPage({
  searchParams,
}: PageProps<"/changement-email/confirmation">) {
  return (
    <Container className="flex justify-center py-12 sm:py-20">
      <div className="w-full max-w-md">
        <div className="border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-6 sm:p-8">
          <h1 className="font-display text-3xl font-bold">Nouvelle adresse email</h1>
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
  searchParams: PageProps<"/changement-email/confirmation">["searchParams"];
}) {
  const { token } = await searchParams;

  if (typeof token !== "string" || !token) {
    return (
      <p className="mt-2 text-muted">
        Ce lien de confirmation est incomplet. Copiez le lien entier depuis le mail reçu, ou
        renouvelez votre demande depuis votre profil.
      </p>
    );
  }

  return (
    <>
      <p className="mt-2 mb-8 text-muted">
        Une demande de changement d&apos;adresse a été faite sur un compte Heros-SOS avec cette
        adresse. Confirmez qu&apos;elle est bien la vôtre : sans cela, la modération ne peut pas
        accepter le changement.
      </p>
      <TokenConfirmationForm
        token={token}
        action={confirmEmailChangeAction}
        submitLabel="Confirmer cette adresse"
        pendingLabel="Confirmation…"
        success={{
          title: "Adresse confirmée.",
          message:
            "La demande est transmise à la modération. L'adresse de connexion du compte ne change qu'après sa validation, et le titulaire en sera prévenu par mail.",
        }}
        invalid={{
          title: "Lien invalide ou expiré.",
          message:
            "Ce lien a expiré, ou la demande a été remplacée ou refusée entre-temps. Si la demande est toujours d'actualité, son titulaire peut la renouveler depuis son profil : un nouveau lien vous sera envoyé.",
        }}
        unavailable="La confirmation est momentanément indisponible. Réessayez dans quelques instants."
        continueHref={routes.home}
        continueLabel="Retour à l'accueil"
      />
    </>
  );
}
