import { addHeroVideoAction } from "@/app/actions/admin";
import { AddVideoForm } from "@/components/hero/AddVideoForm";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { getHeroesForVideos, getHeroVideos, searchImdbTitles } from "@/lib/api";
import { routes } from "@/lib/routes";
import { requireAdminSession } from "@/lib/server-session";
import { apiFailureMessage } from "@/lib/validation/form-state";
import { EmptyState } from "./ModerationFilter";

/** Paramètres d'URL de l'ajout : héros choisi (`heros`) et recherche IMDb (`q`). */
export type AddHeroVideoSearchParams = Promise<{
  heros?: string | string[];
  q?: string | string[];
  statut?: string | string[];
}>;

const first = (value: string | string[] | undefined) =>
  (Array.isArray(value) ? value[0] : value)?.trim() ?? "";

const fieldClasses =
  "min-h-11 w-full min-w-0 rounded-full border border-border bg-background px-4 placeholder:text-muted";

/**
 * Ajout direct d'un film IMDb à la vidéothèque d'un super-héros, publié sans modération.
 * Formulaire GET : fonctionne sans JavaScript et l'URL reste partageable. Lit la session et
 * l'URL : à rendre sous <Suspense>.
 */
export async function AddHeroVideo({ searchParams }: { searchParams: AddHeroVideoSearchParams }) {
  const token = await requireAdminSession();
  const params = await searchParams;
  const query = first(params.q);
  const status = first(params.statut);
  const heroes = await getHeroesForVideos(token);
  const hero = heroes.find(({ id }) => id === first(params.heros));

  const [search, videos] = await Promise.all([
    hero && query ? searchImdbTitles(token, query) : null,
    hero ? getHeroVideos(token) : [],
  ]);
  const owned = new Set(
    videos.filter((video) => video.hero.id === hero?.id).map(({ imdbId }) => imdbId),
  );

  return (
    <details
      open={Boolean(hero)}
      className="mb-8 border-2 border-border bg-surface p-5 shadow-[3px_3px_0_0_var(--color-shadow)] sm:p-6"
    >
      <summary className="cursor-pointer font-display text-lg font-semibold">
        Ajouter un film à un super-héros
      </summary>
      <p className="mt-2 mb-4 text-sm text-muted">
        Le film est publié aussitôt, sans passer par la validation.
      </p>
      <form
        role="search"
        action={routes.adminVideos}
        className="grid gap-3 sm:grid-cols-[minmax(0,14rem)_minmax(0,1fr)_auto]"
      >
        {/* Garde le filtre de la liste de modération affichée en dessous. */}
        {status && <input type="hidden" name="statut" value={status} />}
        <label htmlFor="add-film-hero" className="sr-only">
          Super-héros
        </label>
        <select
          id="add-film-hero"
          name="heros"
          required
          defaultValue={hero?.id ?? ""}
          className={fieldClasses}
        >
          <option value="" disabled>
            Super-héros…
          </option>
          {heroes.map(({ id, alias }) => (
            <option key={id} value={id}>
              {alias}
            </option>
          ))}
        </select>
        <label htmlFor="add-film-q" className="sr-only">
          Titre du film
        </label>
        <input
          id="add-film-q"
          name="q"
          type="search"
          required
          defaultValue={query}
          placeholder="Titre du film"
          className={fieldClasses}
        />
        <button type="submit" className={cn(buttonClasses("secondary"), "shrink-0")}>
          Rechercher
        </button>
      </form>

      {heroes.length === 0 && (
        <p className="mt-3 text-sm text-muted">Aucun super-héros actif pour l&apos;instant.</p>
      )}

      {hero && search && (
        <div className="mt-4">
          {!search.ok ? (
            <p role="alert" className="text-sm text-danger">
              {search.error === "catalog-unavailable"
                ? "Le catalogue IMDb est momentanément indisponible. Réessayez dans quelques instants."
                : apiFailureMessage(search)}
            </p>
          ) : search.titles.length === 0 ? (
            <EmptyState>Aucun film ne correspond à «&nbsp;{query}&nbsp;» sur IMDb.</EmptyState>
          ) : (
            <>
              <p role="status" className="mb-3 text-sm text-muted">
                {search.titles.length} résultat{search.titles.length > 1 ? "s" : ""} sur IMDb, à
                ajouter à la vidéothèque de {hero.alias}
              </p>
              <ul className="space-y-3">
                {search.titles.map((title) => (
                  <li
                    key={title.id}
                    className="flex flex-wrap items-start justify-between gap-4 rounded-2xl border border-border bg-background p-4"
                  >
                    <div className="min-w-0 flex-1">
                      <p className="font-display font-semibold">
                        {title.title}
                        {title.year && (
                          <span className="font-mono text-sm text-muted"> ({title.year})</span>
                        )}
                      </p>
                      {title.plot && (
                        <p className="mt-1 line-clamp-2 text-sm text-pretty text-muted">
                          {title.plot}
                        </p>
                      )}
                    </div>
                    {owned.has(title.id) ? (
                      <p className="text-sm text-muted">Déjà dans sa vidéothèque</p>
                    ) : (
                      <AddVideoForm
                        imdbId={title.id}
                        title={title.title}
                        heroId={hero.id}
                        action={addHeroVideoAction}
                      />
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}
    </details>
  );
}
