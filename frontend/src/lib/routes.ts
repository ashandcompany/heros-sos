import type { UserRole } from "@/types";

/** Routes de l'application. Seules les pages d'accueil, de connexion et de mot de passe oublié existent pour l'instant. */
export const routes = {
  home: "/",
  encyclopedia: "/encyclopedie",
  videos: "/videotheque",
  blog: "/blog",
  login: "/connexion",
  signup: "/inscription",
  forgotPassword: "/mot-de-passe-oublie",
  profile: "/profil",
  heroSpace: "/espace-heros",
  about: "/a-propos",
  contact: "/contact",
  faq: "/faq",
  legal: "/mentions-legales",
  privacy: "/confidentialite",
} as const;

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
};
