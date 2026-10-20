import type { ReactNode } from "react";
import { Container } from "@/components/ui/Container";
import { formatDate } from "@/lib/format";

export interface LegalSection {
  /** Ancre de la section, reprise dans le sommaire. */
  id: string;
  title: string;
  content: ReactNode;
}

interface LegalPageProps {
  title: string;
  intro?: ReactNode;
  /** Date ISO de la dernière mise à jour du texte (`2026-10-20`). */
  updatedAt: string;
  sections: LegalSection[];
}

/** Styles du texte juridique : listes, liens et tableaux, sans composant dédié à chaque balise. */
const contentClasses = [
  "max-w-prose leading-relaxed",
  "[&_a]:text-accent [&_a]:underline [&_strong]:font-semibold",
  "[&_h3]:mt-6 [&_h3]:font-display [&_h3]:text-lg [&_h3]:font-semibold",
  "[&_li]:my-1 [&_p]:my-3 [&_ul]:list-disc [&_ul]:pl-6",
  "[&_table]:my-4 [&_table]:w-full [&_table]:border-collapse [&_table]:text-sm",
  "[&_td]:border-2 [&_td]:border-border [&_td]:p-2 [&_td]:align-top",
  "[&_th]:border-2 [&_th]:border-border [&_th]:bg-surface-raised [&_th]:p-2 [&_th]:text-left",
].join(" ");

/** Gabarit des pages légales : titre, date de mise à jour, sommaire et sections numérotées. */
export function LegalPage({ title, intro, updatedAt, sections }: LegalPageProps) {
  return (
    <Container className="py-10 sm:py-14">
      <header className="max-w-3xl">
        <h1 className="font-display text-3xl font-bold text-balance sm:text-4xl">{title}</h1>
        {intro && <div className="mt-3 text-lg text-pretty text-muted">{intro}</div>}
        <p className="mt-4 font-mono text-sm text-muted">
          Dernière mise à jour : <time dateTime={updatedAt}>{formatDate(updatedAt)}</time>
        </p>
        <div aria-hidden="true" className="rule-thick mt-6 w-24" />
      </header>

      <nav aria-labelledby="legal-toc-title" className="mt-8 max-w-3xl border-2 border-border p-5">
        <h2
          id="legal-toc-title"
          className="font-mono text-xs font-medium tracking-widest text-accent uppercase"
        >
          Sommaire
        </h2>
        <ol className="mt-3 list-decimal space-y-1 pl-6">
          {sections.map((section) => (
            <li key={section.id}>
              <a href={`#${section.id}`} className="hover:text-accent hover:underline">
                {section.title}
              </a>
            </li>
          ))}
        </ol>
      </nav>

      {sections.map((section, index) => (
        <section
          key={section.id}
          id={section.id}
          aria-labelledby={`${section.id}-title`}
          className="mt-12 scroll-mt-20"
        >
          <h2
            id={`${section.id}-title`}
            className="font-display text-2xl font-bold tracking-[0.04em] text-balance uppercase"
          >
            <span className="mr-2 font-mono text-accent">{String(index + 1).padStart(2, "0")}</span>
            {section.title}
          </h2>
          <div className={contentClasses}>{section.content}</div>
        </section>
      ))}
    </Container>
  );
}

/** Tableau défilant horizontalement sur mobile plutôt que de déborder de la page. */
export function LegalTable({ children }: { children: ReactNode }) {
  return (
    <div className="overflow-x-auto">
      <table>{children}</table>
    </div>
  );
}

/**
 * Information que seul le porteur du projet peut fournir. Surlignée pour être repérée (et
 * cherchable avec « À compléter ») avant la mise en ligne.
 */
export function ToComplete({ children }: { children: ReactNode }) {
  return (
    <mark className="bg-mustard px-1 font-mono text-sm text-ink">[À compléter : {children}]</mark>
  );
}
