import type { ReactNode } from "react";
import { Container } from "@/components/ui/Container";

/**
 * Gabarit des pages d'erreur (introuvable, panne) : fiche papier tamponnée, message et actions.
 * Sans état : utilisable depuis `error.tsx`, qui est un Client Component.
 */
export function StatusPage({
  code,
  stamp,
  title,
  children,
  actions,
}: {
  /** Code affiché en tête de fiche (« 404 »). */
  code: string;
  /** Texte du tampon, décoratif. */
  stamp: string;
  title: string;
  children: ReactNode;
  actions: ReactNode;
}) {
  return (
    <Container className="py-14 sm:py-20">
      <article className="panel paper relative mx-auto max-w-2xl p-6 pt-8 sm:p-10">
        <p
          aria-hidden="true"
          className="font-mono text-[0.65rem] tracking-widest text-ink/75 uppercase"
        >
          Rapport d&apos;anomalie — code {code}
        </p>
        <span aria-hidden="true" className="stamp absolute top-6 right-6 hidden sm:inline-block">
          {stamp}
        </span>
        <h1 className="mt-4 font-display text-3xl font-black tracking-[0.04em] text-balance uppercase sm:text-4xl">
          {title}
        </h1>
        <div aria-hidden="true" className="rule-thick mt-4 w-20" />
        <div className="mt-6 space-y-3 text-pretty text-ink/85">{children}</div>
        <div className="mt-8 flex flex-wrap gap-3">{actions}</div>
      </article>
    </Container>
  );
}
