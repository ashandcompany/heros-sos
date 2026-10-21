import type { ApiFailure } from "./api";
import type { ModerationStatus } from "./moderation";
import type { SuperHeroSummary } from "./super-hero";

/** Film ajouté par un super-héros dans sa vidéothèque, publié après modération. */
export interface HeroVideo {
  id: string;
  hero: SuperHeroSummary;
  title: string;
  description: string;
  /** Lien vers la vidéo (hébergeur externe ou fiche IMDb). */
  videoUrl: string;
  /** Identifiant IMDb (`tt…`) quand le film provient du catalogue IMDb. */
  imdbId?: string;
  year?: number;
  /** Date ISO 8601 de l'ajout. */
  addedAt: string;
  moderation: ModerationStatus;
}

/** Film rattaché à la fiche d'un super-vilain par la modération, visible aussitôt. */
export interface VillainFilm {
  id: string;
  villain: { id: string; alias: string };
  title: string;
  description: string;
  /** Lien vers la vidéo ou la fiche IMDb. */
  videoUrl: string;
  imdbId?: string;
  year?: number;
  /** Date ISO 8601 du rattachement. */
  addedAt: string;
}

/** Film du catalogue IMDb. */
export interface ImdbTitle {
  /** Identifiant IMDb, ex : `tt0000001`. */
  id: string;
  title: string;
  year?: number;
  /** Résumé, vide si le catalogue n'en fournit pas. */
  plot: string;
  posterUrl?: string;
}

/** Résultat d'une recherche dans le catalogue IMDb. */
export type ImdbSearchResult =
  | { ok: true; titles: ImdbTitle[] }
  /** Catalogue OMDb injoignable depuis l'API (502). */
  | { ok: false; error: "catalog-unavailable" }
  | ApiFailure;
