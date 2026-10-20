import type { UserRole } from "@/types";

/** Routes de l'application (seules certaines pages existent : voir src/app/). */
export const routes = {
  home: "/",
  encyclopedia: "/encyclopedie",
  videos: "/videotheque",
  blog: "/blog",
  login: "/connexion",
  signup: "/inscription",
  forgotPassword: "/mot-de-passe-oublie",
  /** Lien du mail de réinitialisation (`?token=…`) : doit rester égal à HEROS_PASSWORD_RESET_PATH du backend. */
  passwordReset: "/reinitialisation-mot-de-passe",
  /** Lien du mail d'activation (`?token=…`) : doit rester égal à HEROS_ACTIVATION_PATH du backend. */
  activation: "/activation",
  profile: "/profil",
  profileIncidents: "/profil/incidents",
  profileInterventions: "/profil/interventions",
  heroSpace: "/heros",
  heroDashboard: "/heros/dashboard",
  heroIncidents: "/heros/incidents",
  heroProfile: "/heros/profil",
  heroVideos: "/heros/films",
  heroReputation: "/heros/reputation",
  heroDeclare: "/heros/signaler",
  admin: "/admin",
  adminIncidents: "/admin/incidents",
  adminReportIncident: "/admin/incidents/signaler",
  adminAccounts: "/admin/comptes",
  adminHeroes: "/admin/super-heros",
  adminVideos: "/admin/videotheque",
  adminComments: "/admin/commentaires",
  adminBlog: "/admin/blog",
  adminNewArticle: "/admin/blog/nouveau",
  about: "/a-propos",
  contact: "/contact",
  faq: "/faq",
  legal: "/mentions-legales",
  privacy: "/confidentialite",
} as const;

export const adminIncidentRoute = (id: string) => `${routes.adminIncidents}/${id}`;
export const adminArticleRoute = (id: string) => `${routes.adminBlog}/${id}`;
export const citizenIncidentRoute = (id: string) => `${routes.profileIncidents}/${id}`;
export const heroIncidentRoute = (id: string) => `${routes.heroIncidents}/${id}`;
export const blogArticleRoute = (slug: string) => `${routes.blog}/${slug}`;
export const heroSheetRoute = (id: string) => `${routes.encyclopedia}/super-heros/${id}`;
export const villainSheetRoute = (id: string) => `${routes.encyclopedia}/super-vilains/${id}`;

export interface NavItem {
  label: string;
  href: string;
}

export const mainNav: NavItem[] = [
  { label: "Accueil", href: routes.home },
  { label: "Encyclopédie", href: routes.encyclopedia },
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
  { label: "Signaler un incident", href: routes.heroDeclare },
  { label: "Ma vidéothèque", href: routes.heroVideos },
  { label: "Ma réputation", href: routes.heroReputation },
  { label: "Mon profil", href: routes.heroProfile },
];

/** Sous-navigation du back-office (modération). */
export const adminNav: NavItem[] = [
  { label: "Tableau de bord", href: routes.admin },
  { label: "Incidents", href: routes.adminIncidents },
  { label: "Comptes", href: routes.adminAccounts },
  { label: "Super-héros", href: routes.adminHeroes },
  { label: "Vidéothèque", href: routes.adminVideos },
  { label: "Commentaires", href: routes.adminComments },
  { label: "Blog", href: routes.adminBlog },
];
