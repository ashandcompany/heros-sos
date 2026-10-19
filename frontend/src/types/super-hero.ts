export interface SuperHero {
  id: string;
  /** Nom de super-héros affiché publiquement. */
  alias: string;
  /** URL de la photo de profil. `null` tant qu'aucune photo n'est fournie (placeholder affiché). */
  avatarUrl: string | null;
  /** Pouvoir principal, en une courte phrase. */
  power: string;
  /** Note moyenne attribuée par les citoyens, de 0 à 5. */
  averageRating: number;
  interventionsCount: number;
}

/** Version allégée d'un super-héros, embarquée dans d'autres ressources (ex : un incident). */
export type SuperHeroSummary = Pick<SuperHero, "id" | "alias">;
