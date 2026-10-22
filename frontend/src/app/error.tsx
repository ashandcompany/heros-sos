"use client";

import { useEffect } from "react";
import { StatusPage } from "@/components/layout/StatusPage";
import { ButtonLink, buttonClasses } from "@/components/ui/ButtonLink";
import { routes } from "@/lib/routes";

/**
 * Erreur inattendue dans une page (API injoignable, bug) : en-tête et pied de page restent
 * affichés. En production, le message d'une erreur serveur est masqué par Next ; seul son
 * `digest` est transmis, à rapprocher des journaux du serveur.
 */
export default function Error({ error, retry }: { error: unknown; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  const digest =
    error instanceof Error && "digest" in error && typeof error.digest === "string"
      ? error.digest
      : null;

  return (
    <StatusPage
      code="500"
      stamp="Incident"
      title="Liaison interrompue"
      actions={
        <>
          <button type="button" onClick={() => retry()} className={buttonClasses()}>
            Réessayer
          </button>
          <ButtonLink href={routes.home} variant="secondary">
            Retour à l&apos;accueil
          </ButtonLink>
        </>
      }
    >
      <p>
        Cette page n&apos;a pas pu être chargée. Le service est peut-être momentanément indisponible
        : réessayez dans quelques instants.
      </p>
      {digest && (
        <p className="font-mono text-xs text-ink/75">
          Référence de l&apos;erreur : <span className="select-all">{digest}</span>
        </p>
      )}
    </StatusPage>
  );
}
