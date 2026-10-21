import { Badge } from "@/components/ui/Badge";
import { cn } from "@/components/ui/cn";
import type { VillainSpecialty } from "@/types";

/** Spécialités d'un super-vilain (types d'incident dont il est coutumier). Rien si aucune. */
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
        <li key={specialty.id}>
          <Badge>{specialty.label}</Badge>
        </li>
      ))}
    </ul>
  );
}
