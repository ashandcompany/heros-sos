import type { ReactNode } from "react";

export interface FaqEntry {
  question: string;
  answer: ReactNode;
}

interface FaqCategoryProps {
  id: string;
  /** Code du dossier, affiché en tête (« A », « B »…). */
  code: string;
  title: string;
  entries: FaqEntry[];
}

/**
 * Rubrique de la FAQ : questions dépliables en `<details>` natifs, donc accessibles au clavier
 * et lisibles sans JavaScript.
 */
export function FaqCategory({ id, code, title, entries }: FaqCategoryProps) {
  return (
    <section id={id} aria-labelledby={`${id}-title`} className="scroll-mt-24">
      <h2
        id={`${id}-title`}
        className="flex items-center gap-4 font-display text-2xl font-black tracking-[0.04em] uppercase sm:text-3xl"
      >
        <span
          aria-hidden="true"
          className="inline-flex size-12 shrink-0 items-center justify-center border-[3px] border-edge bg-mustard font-mono text-xl text-ink shadow-[3px_3px_0_0_var(--color-shadow)]"
        >
          {code}
        </span>
        {title}
      </h2>

      <div className="mt-6 space-y-4">
        {entries.map((entry, index) => (
          <details key={entry.question} className="group panel">
            <summary className="flex min-h-14 cursor-pointer list-none items-center gap-4 p-4 hover:bg-surface-raised sm:px-6 [&::-webkit-details-marker]:hidden">
              <span aria-hidden="true" className="font-mono text-xs text-muted">
                {code}.{index + 1}
              </span>
              <span className="flex-1 font-display font-bold tracking-wide">{entry.question}</span>
              <span
                aria-hidden="true"
                className="inline-flex size-8 shrink-0 items-center justify-center border-2 border-edge bg-primary font-mono text-lg leading-none text-primary-foreground transition-transform group-open:rotate-45 motion-reduce:transition-none"
              >
                +
              </span>
            </summary>
            <div className="border-t-2 border-dashed border-border px-4 pt-4 pb-5 text-pretty text-muted sm:px-6 [&_a]:text-accent [&_a]:underline [&_p+p]:mt-3">
              {entry.answer}
            </div>
          </details>
        ))}
      </div>
    </section>
  );
}
