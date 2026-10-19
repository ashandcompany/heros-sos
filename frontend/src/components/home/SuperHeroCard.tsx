import { Avatar } from "@/components/ui/Avatar";
import { RatingStars } from "@/components/ui/RatingStars";
import type { SuperHero } from "@/types";

const interventionsFormatter = new Intl.NumberFormat("fr-FR");

export function SuperHeroCard({ hero }: { hero: SuperHero }) {
  return (
    <article className="flex h-full flex-col items-center rounded-2xl border border-border bg-surface p-5 text-center">
      <Avatar name={hero.alias} src={hero.avatarUrl} size={72} />
      <h3 className="mt-4 font-display text-lg font-semibold">{hero.alias}</h3>
      <p className="mt-1 text-sm text-muted text-pretty">{hero.power}</p>
      <div className="mt-auto flex flex-col items-center gap-1 pt-4">
        <RatingStars rating={hero.averageRating} />
        <p className="text-sm text-muted">
          <span className="font-semibold text-foreground">
            {interventionsFormatter.format(hero.interventionsCount)}
          </span>{" "}
          interventions
        </p>
      </div>
    </article>
  );
}
