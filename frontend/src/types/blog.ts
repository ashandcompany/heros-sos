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
   * Assaini par l'API à l'enregistrement, puis de nouveau avant tout rendu public.
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

/** Article dans une liste : l'API n'y joint pas le contenu. */
export type BlogArticleSummary = Omit<BlogArticle, "content">;

/** Champs saisis dans l'éditeur d'article. */
export interface ArticleInput {
  title: string;
  excerpt: string;
  content: string;
  status: ArticleStatus;
}
