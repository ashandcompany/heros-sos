export type ArticleStatus = "brouillon" | "publie";

export interface BlogArticle {
  id: string;
  title: string;
  /** Identifiant d'URL, dérivé du titre. */
  slug: string;
  /** Chapeau affiché dans les listes. */
  excerpt: string;
  /**
   * Corps de l'article en HTML, produit par l'éditeur du back-office.
   * À assainir côté API avant tout rendu public.
   */
  content: string;
  status: ArticleStatus;
  authorName: string;
  /** Dates ISO 8601. */
  createdAt: string;
  updatedAt: string;
  /** `null` tant que l'article n'a jamais été publié. */
  publishedAt: string | null;
}

/** Champs saisis dans l'éditeur d'article. */
export interface ArticleInput {
  title: string;
  excerpt: string;
  content: string;
  status: ArticleStatus;
}
