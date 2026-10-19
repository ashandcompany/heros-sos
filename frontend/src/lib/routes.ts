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
  profile: "/profil",
  profileIncidents: "/profil/incidents",
  profileInterventions: "/profil/interventions",
  heroSpace: "/espace-heros",
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
  "super-heros": routes.heroSpace,
  admin: routes.admin,
};

/** Sous-navigation de l'espace personnel citoyen. */
export const citizenNav: NavItem[] = [
  { label: "Mon compte", href: routes.profile },
  { label: "Mes incidents", href: routes.profileIncidents },
  { label: "Noter un super-héros", href: routes.profileInterventions },
];

/**
 * Page affichée après connexion. L'espace héros n'existant pas encore,
 * les super-héros restent sur l'accueil.
 */
export const afterLoginRouteByRole: Record<UserRole, string> = {
  citoyen: routes.profile,
  "super-heros": routes.home,
  admin: routes.admin,
};

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
