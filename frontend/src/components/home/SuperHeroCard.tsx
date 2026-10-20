import Link from "next/link";
import { Avatar } from "@/components/ui/Avatar";
import { dossierNumber } from "@/lib/dossier";
import { formatRating } from "@/lib/format";
import { heroSheetRoute } from "@/lib/routes";
import type { SuperHero } from "@/types";

const interventionsFormatter = new Intl.NumberFormat("fr-FR");

/**
 * Fiche d'agent : badge d'identification avec photo N&B, matricule et compteurs numériques.
 * Le pseudo est un lien étiré : toute la carte mène à la fiche détaillée.
 */
export function SuperHeroCard({ hero }: { hero: SuperHero }) {
  return (
    <article className="panel paper flex h-full flex-col p-4 pt-6 text-center focus-within:outline-2 focus-within:outline-offset-4 focus-within:outline-primary">
      {/* Fente du cordon */}
      <span
        aria-hidden="true"
        className="absolute top-2 left-1/2 h-2 w-14 -translate-x-1/2 rounded-full border-2 border-edge bg-background"
      />

      <p
        aria-hidden="true"
        className="font-mono text-[0.65rem] tracking-widest text-ink/75 uppercase"
      >
        Matricule {dossierNumber(hero.id, "AG")}
      </p>

      <div className="mx-auto mt-3 border-4 border-background bg-background shadow-[3px_3px_0_0] shadow-black/40">
        <Avatar name={hero.alias} src={hero.avatarUrl} size={88} square />
      </div>

      <h3 className="mt-3 font-display text-lg font-bold tracking-wider uppercase">
        <Link
          href={heroSheetRoute(hero.id)}
          className="after:absolute after:inset-0 after:z-10 hover:underline focus-visible:outline-none"
        >
          {hero.alias}
        </Link>
      </h3>
      <p className="mt-1 text-sm text-pretty text-ink/85">{hero.power}</p>

      <p className="mx-auto mt-3 inline-flex items-center gap-2 border-2 border-sage bg-sage px-2 py-0.5 font-mono text-[0.7rem] font-medium tracking-widest text-paper uppercase">
        <span aria-hidden="true">✓</span> Agent vérifié
      </p>

      <dl className="mt-auto grid grid-cols-2 gap-3 pt-4">
        <div>
          <dt className="font-mono text-[0.65rem] tracking-widest text-ink/75 uppercase">Note</dt>
          <dd className="mt-1">
            <span aria-hidden="true" className="digital text-lg">
              {formatRating(hero.averageRating)}
            </span>
            <span className="sr-only">{formatRating(hero.averageRating)} sur 5</span>
          </dd>
        </div>
        <div>
          <dt className="font-mono text-[0.65rem] tracking-widest text-ink/75 uppercase">
            Interventions
          </dt>
          <dd className="mt-1">
            <span className="digital text-lg">
              {interventionsFormatter.format(hero.interventionsCount)}
            </span>
          </dd>
        </div>
      </dl>
    </article>
  );
}
