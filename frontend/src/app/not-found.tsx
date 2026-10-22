import type { Metadata } from "next";
import Link from "next/link";
import { StatusPage } from "@/components/layout/StatusPage";
import { ButtonLink } from "@/components/ui/ButtonLink";
import { routes } from "@/lib/routes";

export const metadata: Metadata = { title: "Page introuvable" };

/** Page inconnue, ou fiche inexistante (`notFound()` : héros, vilain, incident, article…). */
export default function NotFound() {
  return (
    <StatusPage
      code="404"
      stamp="Introuvable"
      title="Dossier introuvable"
      actions={
        <>
          <ButtonLink href={routes.home}>Retour à l&apos;accueil</ButtonLink>
          <ButtonLink href={routes.encyclopedia} variant="secondary">
            Consulter l&apos;encyclopédie
          </ButtonLink>
        </>
      }
    >
      <p>
        Cette page n&apos;existe pas ou plus : le lien est peut-être ancien, ou la fiche a été
        retirée.
      </p>
      <p>
        Vous pensez qu&apos;il s&apos;agit d&apos;une erreur&nbsp;?{" "}
        <Link href={routes.contact} className="text-accent underline">
          Signalez-la-nous
        </Link>
        .
      </p>
    </StatusPage>
  );
}
