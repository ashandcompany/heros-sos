"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Suspense, useEffect, useId, useRef, useState } from "react";
import { cn } from "@/components/ui/cn";
import { isNavGroup, type NavEntry, type NavItem } from "@/lib/routes";

/** Lien actif : le plus long `href` qui correspond à la page (ou à l'une de ses parentes). */
function currentHref(entries: NavEntry[], pathname: string): string | undefined {
  return entries
    .flatMap((entry) => (isNavGroup(entry) ? entry.items : [entry]))
    .filter(({ href }) => pathname === href || pathname.startsWith(`${href}/`))
    .sort((a, b) => b.href.length - a.href.length)[0]?.href;
}

interface SpaceNavProps {
  items: NavEntry[];
  label: string;
}

/**
 * Sous-navigation d'un espace réservé. Les liens simples défilent horizontalement sur mobile ; une
 * navigation avec des groupes (`NavGroup`) les affiche en menus déroulants et passe à la ligne.
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

const tabClasses = (isCurrent: boolean) =>
  cn(
    "-mb-px inline-flex min-h-11 items-center gap-1.5 border-b-2 px-3 text-sm font-medium whitespace-nowrap",
    isCurrent
      ? "border-primary text-foreground"
      : "border-transparent text-muted hover:text-foreground",
  );

function ChevronIcon({ open }: { open: boolean }) {
  return (
    <svg
      viewBox="0 0 24 24"
      aria-hidden="true"
      className={cn(
        "size-4 fill-none stroke-current transition-transform motion-reduce:transition-none",
        open && "rotate-180",
      )}
    >
      <path d="m6 9 6 6 6-6" strokeWidth="2" strokeLinecap="square" />
    </svg>
  );
}

function SpaceNavView({ items, label, pathname }: SpaceNavProps & { pathname: string | null }) {
  const current = pathname ? currentHref(items, pathname) : undefined;
  const grouped = items.some(isNavGroup);
  // Un seul menu ouvert à la fois : on garde le libellé de son groupe.
  const [openGroup, setOpenGroup] = useState<string | null>(null);
  const navRef = useRef<HTMLElement>(null);

  // Fermé par Échap (le focus revient sur son bouton) ou par un clic en dehors de la navigation.
  useEffect(() => {
    if (!openGroup) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpenGroup(null);
      navRef.current?.querySelector<HTMLButtonElement>("[aria-expanded=true]")?.focus();
    };
    const handlePointerDown = (event: PointerEvent) => {
      if (!navRef.current?.contains(event.target as Node)) setOpenGroup(null);
    };

    document.addEventListener("keydown", handleKeyDown);
    document.addEventListener("pointerdown", handlePointerDown);
    return () => {
      document.removeEventListener("keydown", handleKeyDown);
      document.removeEventListener("pointerdown", handlePointerDown);
    };
  }, [openGroup]);

  const ariaCurrent = (href: string) =>
    href === current ? (pathname === href ? "page" : "true") : undefined;

  const renderLink = (item: NavItem) => (
    <li key={item.href}>
      <Link
        href={item.href}
        aria-current={ariaCurrent(item.href)}
        className={tabClasses(item.href === current)}
      >
        {item.label}
      </Link>
    </li>
  );

  return (
    <nav
      ref={navRef}
      aria-label={label}
      className={cn("mb-8", grouped ? "relative" : "-mx-4 overflow-x-auto px-4 sm:mx-0 sm:px-0")}
    >
      <ul
        className={cn(
          "flex gap-2 border-b border-border",
          grouped ? "flex-wrap" : "w-max sm:w-auto",
        )}
      >
        {items.map((entry) =>
          isNavGroup(entry) ? (
            <NavDropdown
              key={entry.label}
              group={entry}
              open={openGroup === entry.label}
              isCurrent={entry.items.some(({ href }) => href === current)}
              ariaCurrent={ariaCurrent}
              onToggle={() => setOpenGroup((open) => (open === entry.label ? null : entry.label))}
              onClose={() => setOpenGroup(null)}
            />
          ) : (
            renderLink(entry)
          ),
        )}
      </ul>
    </nav>
  );
}

interface NavDropdownProps {
  group: { label: string; items: NavItem[] };
  open: boolean;
  /** La page courante est l'un des liens du groupe. */
  isCurrent: boolean;
  ariaCurrent: (href: string) => "page" | "true" | undefined;
  onToggle: () => void;
  onClose: () => void;
}

/**
 * Bouton qui déplie la liste des liens d'un groupe (motif « disclosure » : de simples liens, pas un
 * menu ARIA). Sur mobile, la liste occupe toute la largeur de la navigation ; dès `sm`, elle se
 * place sous son bouton.
 */
function NavDropdown({ group, open, isCurrent, ariaCurrent, onToggle, onClose }: NavDropdownProps) {
  const listId = useId();

  return (
    <li
      className="sm:relative"
      // Le focus quitte le groupe (tabulation) : la liste se referme.
      onBlur={(event) => {
        if (open && !event.currentTarget.contains(event.relatedTarget)) onClose();
      }}
    >
      <button
        type="button"
        aria-expanded={open}
        aria-controls={listId}
        onClick={onToggle}
        className={tabClasses(isCurrent || open)}
      >
        {group.label}
        <ChevronIcon open={open} />
      </button>
      <ul
        id={listId}
        hidden={!open}
        className="absolute inset-x-0 top-full z-20 mt-1 border-2 border-edge bg-surface py-1 shadow-[3px_3px_0_0_var(--color-shadow)] sm:right-auto sm:min-w-56"
      >
        {group.items.map((item) => {
          const isItemCurrent = ariaCurrent(item.href) !== undefined;
          return (
            <li key={item.href}>
              <Link
                href={item.href}
                onClick={onClose}
                aria-current={ariaCurrent(item.href)}
                className={cn(
                  "flex min-h-11 items-center border-l-4 px-4 text-sm font-medium",
                  isItemCurrent
                    ? "border-primary bg-surface-raised text-foreground"
                    : "border-transparent text-muted hover:bg-surface-raised hover:text-foreground",
                )}
              >
                {item.label}
              </Link>
            </li>
          );
        })}
      </ul>
    </li>
  );
}
