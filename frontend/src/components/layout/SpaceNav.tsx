"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Suspense } from "react";
import { cn } from "@/components/ui/cn";
import type { NavItem } from "@/lib/routes";

/** Lien actif : le plus long `href` qui correspond à la page (ou à l'une de ses parentes). */
function currentHref(items: NavItem[], pathname: string): string | undefined {
  return items
    .filter(({ href }) => pathname === href || pathname.startsWith(`${href}/`))
    .sort((a, b) => b.href.length - a.href.length)[0]?.href;
}

interface SpaceNavProps {
  items: NavItem[];
  label: string;
}

/**
 * Sous-navigation d'un espace réservé (défilement horizontal sur mobile).
 * Sur une route à paramètre dynamique, la page courante n'est connue qu'à la requête :
 * la navigation est d'abord rendue sans lien actif (fallback), puis mise à jour.
 */
export function SpaceNav(props: SpaceNavProps) {
  return (
    <Suspense fallback={<SpaceNavView {...props} pathname={null} />}>
      <SpaceNavWithPathname {...props} />
    </Suspense>
  );
}

function SpaceNavWithPathname(props: SpaceNavProps) {
  return <SpaceNavView {...props} pathname={usePathname()} />;
}

function SpaceNavView({ items, label, pathname }: SpaceNavProps & { pathname: string | null }) {
  const current = pathname ? currentHref(items, pathname) : undefined;

  return (
    <nav aria-label={label} className="-mx-4 mb-8 overflow-x-auto px-4 sm:mx-0 sm:px-0">
      <ul className="flex w-max gap-2 border-b border-border sm:w-auto">
        {items.map((item) => {
          const isCurrent = item.href === current;
          return (
            <li key={item.href}>
              <Link
                href={item.href}
                aria-current={isCurrent ? (pathname === item.href ? "page" : "true") : undefined}
                className={cn(
                  "-mb-px inline-flex min-h-11 items-center border-b-2 px-3 text-sm font-medium whitespace-nowrap",
                  isCurrent
                    ? "border-primary text-foreground"
                    : "border-transparent text-muted hover:text-foreground",
                )}
              >
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
