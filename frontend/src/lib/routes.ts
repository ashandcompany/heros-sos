/** Routes de l'application. Seule la page d'accueil existe pour l'instant. */
export const routes = {
  home: "/",
  encyclopedia: "/encyclopedie",
  videos: "/videotheque",
  blog: "/blog",
  login: "/connexion",
  signup: "/inscription",
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
