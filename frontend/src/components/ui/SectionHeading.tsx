import { cn } from "./cn";

interface SectionHeadingProps {
  id: string;
  eyebrow?: string;
  title: string;
  description?: string;
  className?: string;
}

/** Titre de section (h2). L'`id` sert de cible à `aria-labelledby` sur la section parente. */
export function SectionHeading({
  id,
  eyebrow,
  title,
  description,
  className,
}: SectionHeadingProps) {
  return (
    <div className={cn("max-w-2xl", className)}>
      {eyebrow && (
        <p className="mb-2 text-sm font-semibold tracking-widest text-accent uppercase">
          {eyebrow}
        </p>
      )}
      <h2 id={id} className="font-display text-2xl font-bold text-balance sm:text-3xl">
        {title}
      </h2>
      {description && <p className="mt-3 text-muted text-pretty">{description}</p>}
    </div>
  );
}
