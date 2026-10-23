import type { UserRole } from "@/types";

/** Routes de l'application (seules certaines pages existent : voir src/app/). */
export const routes = {
  home: "/",
  encyclopedia: "/encyclopedie",
  map: "/carte",
  videos: "/videotheque",
  blog: "/blog",
  login: "/connexion",
  signup: "/inscription",
  /** Saisie du code de double authentification, après le mot de passe (session limitée). */
  loginVerification: "/connexion/verification",
  /** Activation obligatoire de la double authentification (modérateur en production, session limitée). */
  loginTwoFactorSetup: "/connexion/double-authentification",
  forgotPassword: "/mot-de-passe-oublie",
  /** Lien du mail de réinitialisation (`?token=…`) : doit rester égal à HEROS_PASSWORD_RESET_PATH du backend. */
  passwordReset: "/reinitialisation-mot-de-passe",
  /** Lien du mail d'activation (`?token=…`) : doit rester égal à HEROS_ACTIVATION_PATH du backend. */
  activation: "/activation",
  /** Lien du mail de confirmation de la newsletter (`?token=…`) : doit rester égal à HEROS_NEWSLETTER_CONFIRM_PATH du backend. */
  newsletterConfirmation: "/newsletter/confirmation",
  /** Lien de désinscription de chaque newsletter (`?token=…`) : doit rester égal à HEROS_NEWSLETTER_UNSUBSCRIBE_PATH du backend. */
  newsletterUnsubscribe: "/newsletter/desinscription",
  /** Lien du mail envoyé à la nouvelle adresse d'un compte (`?token=…`) : doit rester égal à HEROS_EMAIL_CHANGE_CONFIRM_PATH du backend. */
  emailChangeConfirmation: "/changement-email/confirmation",
  profile: "/profil",
  profileIncidents: "/profil/incidents",
  profileInterventions: "/profil/interventions",
  heroSpace: "/heros",
  heroDashboard: "/heros/dashboard",
  heroIncidents: "/heros/incidents",
  heroMap: "/heros/carte",
  heroProfile: "/heros/profil",
  heroVideos: "/heros/films",
  heroReputation: "/heros/reputation",
  heroDeclare: "/heros/signaler",
  heroRate: "/heros/noter",
  admin: "/admin",
  adminIncidents: "/admin/incidents",
  adminReportIncident: "/admin/incidents/signaler",
  adminRate: "/admin/noter",
  adminIncidentTypes: "/admin/types-incident",
  adminVillains: "/admin/super-vilains",
  adminAccounts: "/admin/comptes",
  adminHeroes: "/admin/super-heros",
  adminVideos: "/admin/videotheque",
  adminComments: "/admin/commentaires",
  adminProfile: "/admin/profil",
  adminSecurity: "/admin/securite",
  adminBlog: "/admin/blog",
  adminNewArticle: "/admin/blog/nouveau",
  about: "/a-propos",
  contact: "/contact",
  faq: "/faq",
  legal: "/mentions-legales",
  privacy: "/confidentialite",
} as const;

export const adminIncidentRoute = (id: string) => `${routes.adminIncidents}/${id}`;
export const adminVillainRoute = (id: string) => `${routes.adminVillains}/${id}`;
export const adminArticleRoute = (id: string) => `${routes.adminBlog}/${id}`;
export const citizenIncidentRoute = (id: string) => `${routes.profileIncidents}/${id}`;
export const heroIncidentRoute = (id: string) => `${routes.heroIncidents}/${id}`;
export const blogArticleRoute = (slug: string) => `${routes.blog}/${slug}`;
export const heroSheetRoute = (id: string) => `${routes.encyclopedia}/super-heros/${id}`;
export const villainSheetRoute = (id: string) => `${routes.encyclopedia}/super-vilains/${id}`;
/** Onglet des super-vilains de l'encyclopédie (celui des héros est `routes.encyclopedia`). */
export const encyclopediaVillainsRoute = `${routes.encyclopedia}?onglet=vilains`;

export interface NavItem {
  label: string;
  href: string;
}

export const mainNav: NavItem[] = [
  { label: "Accueil", href: routes.home },
  { label: "Encyclopédie", href: routes.encyclopedia },
  { label: "Carte", href: routes.map },
  { label: "Vidéothèque", href: routes.videos },
  { label: "Blog", href: routes.blog },
];

/** Page d'accueil de l'espace personnel selon le rôle de l'utilisateur connecté. */
export const accountHomeByRole: Record<UserRole, string> = {
  citoyen: routes.profile,
  "super-heros": routes.heroDashboard,
  admin: routes.admin,
};

/** Sous-navigation de l'espace personnel citoyen. */
export const citizenNav: NavItem[] = [
  { label: "Mon compte", href: routes.profile },
  { label: "Mes incidents", href: routes.profileIncidents },
  { label: "Noter un super-héros", href: routes.profileInterventions },
];

/** Sous-navigation de l'espace professionnel des super-héros. */
export const heroNav: NavItem[] = [
  { label: "Tableau de bord", href: routes.heroDashboard },
  { label: "Incidents", href: routes.heroIncidents },
  { label: "Carte", href: routes.heroMap },
  { label: "Signaler un incident", href: routes.heroDeclare },
  { label: "Noter un héros", href: routes.heroRate },
  { label: "Ma vidéothèque", href: routes.heroVideos },
  { label: "Ma réputation", href: routes.heroReputation },
  { label: "Mon profil", href: routes.heroProfile },
];

/** Sous-navigation du back-office (modération). */
export const adminNav: NavItem[] = [
  { label: "Tableau de bord", href: routes.admin },
  { label: "Incidents", href: routes.adminIncidents },
  { label: "Types d'incident", href: routes.adminIncidentTypes },
  { label: "Comptes", href: routes.adminAccounts },
  { label: "Super-héros", href: routes.adminHeroes },
  { label: "Super-vilains", href: routes.adminVillains },
  { label: "Vidéothèque", href: routes.adminVideos },
  { label: "Commentaires", href: routes.adminComments },
  { label: "Noter un héros", href: routes.adminRate },
  { label: "Blog", href: routes.adminBlog },
  { label: "Mon profil", href: routes.adminProfile },
  { label: "Sécurité", href: routes.adminSecurity },
];
