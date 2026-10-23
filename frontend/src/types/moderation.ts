/**
 * Décision de la modération sur un contenu ou un compte :
 * en attente de vérification, validé, ou rejeté (incident jugé faux, compte suspendu,
 * commentaire masqué, film refusé…).
 */
export type ModerationStatus = "en-attente" | "valide" | "rejete";

/** Nombre d'éléments en attente de modération, par section du back-office. */
export interface ModerationSummary {
  incidents: number;
  emailChanges: number;
  heroApplications: number;
  videos: number;
  /** Réponses de super-héros en attente de vérification (les avis sont publiés d'office). */
  comments: number;
  draftArticles: number;
  /** Numéros de la newsletter en brouillon, à envoyer. */
  draftNewsletters: number;
}
