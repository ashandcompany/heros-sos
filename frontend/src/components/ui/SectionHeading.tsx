import { cn } from "./cn";

interface SectionHeadingProps {
  id: string;
  eyebrow?: string;
  /** Numéro de section administrative (« 02 » → « SECTION 02 »). */
  section?: string;
  title: string;
  description?: string;
  className?: string;
}

/** Titre de section (h2). L'`id` sert de cible à `aria-labelledby` sur la section parente. */
export function SectionHeading({
  id,
  eyebrow,
  section,
  title,
  description,
  className,
}: SectionHeadingProps) {
  const label = [section && `Section ${section}`, eyebrow].filter(Boolean).join(" — ");

  return (
    <div className={cn("max-w-3xl", className)}>
      {label && (
        <p className="mb-3 inline-block bg-accent px-2 py-0.5 font-mono text-xs font-semibold tracking-widest text-background uppercase">
          {label}
        </p>
      )}
      <h2
        id={id}
        className="font-display text-3xl font-black tracking-[0.04em] text-balance uppercase sm:text-4xl"
      >
        {title}
      </h2>
      <div aria-hidden="true" className="rule-thick mt-4 w-20" />
      {description && <p className="mt-4 text-muted text-pretty">{description}</p>}
    </div>
  );
}
