import { cn } from "@/components/ui/cn";

/** Initiales du pseudo (« Le Docteur Néant » → « DN »), articles ignorés. */
function initials(alias: string): string {
  const words = alias.split(/[\s-]+/).filter((word) => !/^(l[ea']?|les|the)$/i.test(word));
  return (words.length > 0 ? words : [alias])
    .slice(0, 2)
    .map((word) => word.replace(/^l'/i, "").charAt(0).toUpperCase())
    .join("");
}

/** Photo d'identité judiciaire décorative : les vilains n'ont pas de portrait. */
export function VillainMugshot({
  alias,
  size,
  className,
}: {
  alias: string;
  size: number;
  className?: string;
}) {
  return (
    <span
      aria-hidden="true"
      className={cn("mugshot shrink-0", className)}
      style={{ width: size, fontSize: size * 0.36 }}
    >
      {initials(alias)}
    </span>
  );
}
