import type { VillainFilm } from "@/types";

/** Films rattachés aux fiches des super-vilains (titres IMDb réels, rattachements fictifs). */
export const villainFilmsMock: VillainFilm[] = [
  {
    id: "villain-film-1",
    villain: { id: "villain-1", alias: "Le Corbeau Gris" },
    title: "The Birds",
    description: "A wealthy San Francisco socialite pursues a potential boyfriend to a small town.",
    videoUrl: "https://www.imdb.com/title/tt0056869/",
    imdbId: "tt0056869",
    year: 1963,
    addedAt: "2026-10-02T14:00:00+02:00",
  },
];
