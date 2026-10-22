import Image from "next/image";
import { cn } from "./cn";

const gradients = [
  "from-primary/80 to-accent/60",
  "from-accent/80 to-success/60",
  "from-success/80 to-primary/60",
  "from-rating/80 to-primary/60",
];

/** Préfixe des routes de l'API sur l'origine du site (voir le Caddyfile du reverse proxy). */
const API_PATH_PREFIX = "/api/v1/";

function hashString(value: string): number {
  let hash = 0;
  for (const char of value) hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  return hash;
}

/** « Jean-Marc Dupont » → « JD », « Batman » → « BA ». */
function initialsOf(name: string): string {
  const words = name.trim().split(/\s+/).filter(Boolean);
  const initials = words.length > 1 ? words[0][0] + words[1][0] : (words[0] ?? "").slice(0, 2);
  return initials.toUpperCase() || "?";
}

interface AvatarProps {
  name: string;
  src: string | null;
  size?: number;
  className?: string;
  /** Photo d'identité carrée en noir et blanc (fiches d'agent). */
  square?: boolean;
}

/**
 * Photo de profil. Sans `src`, affiche un placeholder avec les initiales.
 * Décoratif (alt vide) : le nom est toujours affiché à côté.
 */
export function Avatar({ name, src, size = 80, className, square = false }: AvatarProps) {
  const classes = cn(
    "shrink-0 overflow-hidden",
    square ? "rounded-none grayscale contrast-110" : "rounded-full",
    className,
  );

  if (src) {
    return (
      <Image
        src={src}
        alt=""
        width={size}
        height={size}
        // Photo envoyée par l'utilisateur, déjà recadrée à l'envoi (voir cropAvatar) et ré-encodée
        // par l'API : pas d'optimisation nécessaire. Elle n'est pas possible pour une data URL
        // (mocks), ni pour une photo de l'API (`/api/v1/avatars/…`) : l'optimiseur de Next la
        // chercherait sur son propre serveur, alors que le reverse proxy l'envoie à l'API.
        unoptimized={src.startsWith("data:") || src.startsWith(API_PATH_PREFIX)}
        className={cn(classes, "object-cover")}
      />
    );
  }

  const initials = initialsOf(name);
  const gradient = square
    ? "from-surface-raised to-background"
    : gradients[hashString(name) % gradients.length];

  return (
    <span
      aria-hidden="true"
      style={{ width: size, height: size, fontSize: Math.round(size * 0.34) }}
      className={cn(
        classes,
        "inline-flex items-center justify-center bg-linear-to-br font-display font-bold text-paper ring-2 ring-border",
        gradient,
      )}
    >
      {initials}
    </span>
  );
}
