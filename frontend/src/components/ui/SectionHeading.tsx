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
        <p className="mb-2 font-mono text-sm font-medium tracking-widest text-accent uppercase">
          {label}
        </p>
      )}
      <h2
        id={id}
        className="font-display text-2xl font-bold tracking-[0.12em] text-balance uppercase sm:text-3xl"
      >
        {title}
      </h2>
      <div aria-hidden="true" className="rule-thick mt-4 w-24" />
      {description && <p className="mt-4 text-muted text-pretty">{description}</p>}
    </div>
  );
}
