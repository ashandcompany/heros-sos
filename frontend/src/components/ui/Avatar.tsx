import Image from "next/image";
import { cn } from "./cn";

const gradients = [
  "from-primary/80 to-accent/60",
  "from-accent/80 to-success/60",
  "from-success/80 to-primary/60",
  "from-rating/80 to-primary/60",
];

function hashString(value: string): number {
  let hash = 0;
  for (const char of value) hash = (hash * 31 + char.charCodeAt(0)) >>> 0;
  return hash;
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
        // Photo envoyée par l'utilisateur (data URL dans les mocks) : pas d'optimisation possible.
        unoptimized={src.startsWith("data:")}
        className={cn(classes, "object-cover")}
      />
    );
  }

  const initials = name.slice(0, 2).toUpperCase();
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
