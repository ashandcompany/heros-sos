import { addVillainFilmAction, deleteVillainFilmAction } from "@/app/actions/admin";
import { AddVideoForm } from "@/components/hero/AddVideoForm";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { getVillainFilms, searchImdbTitles } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { adminVillainRoute } from "@/lib/routes";
import { apiFailureMessage } from "@/lib/validation/form-state";
import type { Villain } from "@/types";
import { ConfirmDeleteButton } from "./ConfirmDeleteButton";
import { EmptyState } from "./ModerationFilter";

const fieldClasses =
  "min-h-11 w-full min-w-0 rounded-full border border-border bg-background px-4 placeholder:text-muted";

/**
 * Films rattachés à un super-vilain : liste, retrait, et ajout depuis IMDb (visible aussitôt).
 * Recherche par formulaire GET (`?q=`) : fonctionne sans JavaScript.
 */
export async function VillainFilms({
  token,
  villain,
  query,
}: {
  token: string;
  villain: Pick<Villain, "id" | "alias">;
  query: string;
}) {
  const [films, search] = await Promise.all([
    getVillainFilms(villain.id),
    query ? searchImdbTitles(token, query) : null,
  ]);
  const linked = new Set(films.map(({ imdbId }) => imdbId));

  return (
    <section
      id="films"
      aria-labelledby="films-title"
      className="mt-6 border-2 border-border bg-surface shadow-[3px_3px_0_0_var(--color-shadow)] p-5 sm:p-8"
    >
      <h2 id="films-title" className="font-display text-xl font-semibold">
        Films
      </h2>
      <p className="mt-1 mb-6 text-sm text-muted">
        Films où apparaît le super-vilain, affichés sur sa fiche publique dès leur ajout.
      </p>

      {films.length === 0 ? (
        <EmptyState>Aucun film rattaché pour l&apos;instant.</EmptyState>
      ) : (
        <ul className="space-y-3">
          {films.map((film) => (
            <li
              key={film.id}
              className="flex flex-wrap items-start justify-between gap-4 rounded-2xl border border-border bg-background p-4"
            >
              <div className="min-w-0 flex-1">
                <p className="font-display font-semibold">
                  {film.title}
                  {film.year && (
                    <span className="font-mono text-sm text-muted"> ({film.year})</span>
                  )}
                </p>
                <p className="mt-1 text-xs text-muted">
                  Ajouté le <time dateTime={film.addedAt}>{formatDate(film.addedAt)}</time>
                </p>
              </div>
              <ConfirmDeleteButton
                action={deleteVillainFilmAction}
                id={`${villain.id}/${film.id}`}
                subject={film.title}
                description={`Le film sera retiré de la fiche de ${villain.alias}.`}
              />
            </li>
          ))}
        </ul>
      )}

      <form
        role="search"
        action={`${adminVillainRoute(villain.id)}#films`}
        className="mt-6 flex flex-col gap-3 sm:flex-row"
      >
        <label htmlFor="villain-film-q" className="sr-only">
          Titre du film
        </label>
        <input
          id="villain-film-q"
          name="q"
          type="search"
          required
          defaultValue={query}
          placeholder="Rechercher un film sur IMDb"
          className={fieldClasses}
        />
        <button type="submit" className={cn(buttonClasses("secondary"), "shrink-0")}>
          Rechercher
        </button>
      </form>

      {search && (
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
                {search.titles.length} résultat{search.titles.length > 1 ? "s" : ""} sur IMDb
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
                    {linked.has(title.id) ? (
                      <p className="text-sm text-muted">Déjà sur sa fiche</p>
                    ) : (
                      <AddVideoForm
                        imdbId={title.id}
                        title={title.title}
                        villainId={villain.id}
                        action={addVillainFilmAction}
                      />
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      )}
    </section>
  );
}
