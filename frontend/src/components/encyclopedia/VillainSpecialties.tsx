import { cn } from "@/components/ui/cn";
import type { VillainSpecialty } from "@/types";

/**
 * Spécialités d'un super-vilain (types d'incident dont il est coutumier), en étiquettes encrées
 * à poser sur une fiche papier. Rien si aucune.
 */
export function VillainSpecialties({
  specialties,
  className,
}: {
  specialties: VillainSpecialty[];
  className?: string;
}) {
  if (specialties.length === 0) return null;
  return (
    <ul aria-label="Spécialités" className={cn("flex flex-wrap gap-1.5", className)}>
      {specialties.map((specialty) => (
        <li
          key={specialty.id}
          className="border border-dashed border-ink/60 px-1.5 py-0.5 font-mono text-[0.7rem] font-medium tracking-wider text-ink/85 uppercase"
        >
          {specialty.label}
        </li>
      ))}
    </ul>
  );
}
