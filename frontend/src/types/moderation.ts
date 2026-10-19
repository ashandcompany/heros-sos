/**
 * Décision de la modération sur un contenu ou un compte :
 * en attente de vérification, validé, ou rejeté (incident jugé faux, compte suspendu,
 * commentaire masqué, film refusé…).
 */
export type ModerationStatus = "en-attente" | "valide" | "rejete";

/** Nombre d'éléments en attente de modération, par section du back-office. */
export interface ModerationSummary {
  incidents: number;
  accounts: number;
  emailChanges: number;
  heroApplications: number;
  videos: number;
  comments: number;
  draftArticles: number;
}
