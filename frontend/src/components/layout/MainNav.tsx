"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { Suspense, useEffect, useId, useRef, useState } from "react";
import { Avatar } from "@/components/ui/Avatar";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { logoutAction } from "@/app/actions/auth";
import { mainNav, routes } from "@/lib/routes";
import type { AvatarData } from "@/lib/api";
import type { AuthUser } from "@/types";

/**
 * Utilisateur connecté (`null` : visiteur, `undefined` : session en cours de lecture).
 * `avatar` est `null` pour un compte sans profil (administrateur) : initiales du `displayName`.
 */
type NavUser =
  (Pick<AuthUser, "displayName" | "role"> & { avatar: AvatarData | null }) | null | undefined;

function MenuIcon({ open }: { open: boolean }) {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className="size-6 fill-none stroke-current">
      {open ? (
        <path d="M6 6l12 12M18 6 6 18" strokeWidth="2" strokeLinecap="round" />
      ) : (
        <path d="M4 7h16M4 12h16M4 17h16" strokeWidth="2" strokeLinecap="round" />
      )}
    </svg>
  );
}

function LogoutIcon() {
  return (
    <svg viewBox="0 0 24 24" aria-hidden="true" className="size-5 fill-none stroke-current">
      <path
        d="M14 4H5v16h9M10 12h10m-4-4 4 4-4 4"
        strokeWidth="2"
        strokeLinecap="square"
        strokeLinejoin="miter"
      />
    </svg>
  );
}

/** Lien vers l'espace de chaque rôle. */
const spaceLinkByRole: Record<AuthUser["role"], { label: string; href: string } | null> = {
  citoyen: { label: "Mon espace", href: routes.profile },
  "super-heros": { label: "Espace pro", href: routes.heroDashboard },
  admin: { label: "Back-office", href: routes.admin },
};

interface AccountActionsProps {
  user: NavUser;
  variant: "desktop" | "mobile";
  isSpacePage: boolean;
  onNavigate: () => void;
}

/** Boutons de compte : connexion / inscription, ou utilisateur connecté et déconnexion. */
function AccountActions({ user, variant, isSpacePage, onNavigate }: AccountActionsProps) {
  if (user === undefined) return null;

  const mobile = variant === "mobile";
  // Sur desktop, boutons compacts pour que la barre tienne même avec un compte connecté.
  const size = mobile ? "md" : "sm";

  if (!user) {
    return (
      <>
        <Link
          href={routes.login}
          onClick={onNavigate}
          className={buttonClasses(mobile ? "secondary" : "ghost", size)}
        >
          Connexion
        </Link>
        <Link href={routes.signup} onClick={onNavigate} className={buttonClasses("primary", size)}>
          Inscription
        </Link>
      </>
    );
  }

  const spaceLink = spaceLinkByRole[user.role];

  return (
    <>
      {/* Sur desktop étroit, seule la photo reste : le nom passe en infobulle. */}
      <p
        title={mobile ? undefined : user.displayName}
        className={cn(
          "flex min-w-0 items-center gap-2 text-sm text-muted",
          mobile && "col-span-2 gap-3",
        )}
      >
        <Avatar
          name={user.avatar?.name ?? user.displayName}
          src={user.avatar?.src ?? null}
          size={mobile ? 40 : 32}
        />
        {mobile ? (
          <span>
            Connecté·e : <span className="font-semibold text-foreground">{user.displayName}</span>
          </span>
        ) : (
          <span className="sr-only max-w-32 truncate font-semibold text-foreground xl:not-sr-only">
            <span className="sr-only">Connecté·e : </span>
            {user.displayName}
          </span>
        )}
      </p>
      {spaceLink && (
        <Link
          href={spaceLink.href}
          onClick={onNavigate}
          aria-current={isSpacePage ? "page" : undefined}
          className={buttonClasses("secondary", size)}
        >
          {spaceLink.label}
        </Link>
      )}
      <form action={logoutAction} className={cn(mobile && !spaceLink && "col-span-2")}>
        {mobile ? (
          <button type="submit" className={cn(buttonClasses("secondary"), "w-full")}>
            Déconnexion
          </button>
        ) : (
          <button
            type="submit"
            title="Déconnexion"
            aria-label="Déconnexion"
            className="inline-flex size-9 items-center justify-center text-muted hover:bg-surface-raised hover:text-accent"
          >
            <LogoutIcon />
          </button>
        )}
      </form>
    </>
  );
}

/**
 * Navigation principale : liens en ligne sur desktop, menu burger sur mobile.
 * Sur une route à paramètre dynamique, la page courante n'est connue qu'à la requête :
 * la navigation est d'abord rendue sans lien actif (fallback), puis mise à jour.
 */
export function MainNav({ user }: { user: NavUser }) {
  return (
    <Suspense fallback={<MainNavView user={user} pathname={null} />}>
      <MainNavWithPathname user={user} />
    </Suspense>
  );
}

function MainNavWithPathname({ user }: { user: NavUser }) {
  return <MainNavView user={user} pathname={usePathname()} />;
}

function MainNavView({ user, pathname }: { user: NavUser; pathname: string | null }) {
  const [open, setOpen] = useState(false);
  const menuId = useId();
  const toggleRef = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    if (!open) return;

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        setOpen(false);
        toggleRef.current?.focus();
      }
    };

    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [open]);

  const close = () => setOpen(false);
  const isSpacePage =
    !!pathname &&
    ["/admin", "/profil", routes.heroSpace].some(
      (prefix) => pathname === prefix || pathname.startsWith(`${prefix}/`),
    );

  const navLinks = (variant: "desktop" | "mobile") =>
    mainNav.map((item) => {
      const isCurrent = pathname === item.href;
      return (
        <li key={item.href}>
          <Link
            href={item.href}
            onClick={close}
            aria-current={isCurrent ? "page" : undefined}
            className={cn(
              "block font-mono font-medium tracking-wider uppercase",
              variant === "desktop"
                ? "border-2 border-b-0 px-3 py-2.5 text-xs font-semibold"
                : "px-3 py-3 text-sm",
              variant === "desktop" &&
                (isCurrent
                  ? "border-edge bg-primary text-primary-foreground"
                  : "border-border bg-surface text-muted hover:border-edge hover:bg-surface-raised hover:text-foreground"),
              variant === "mobile" &&
                (isCurrent ? "bg-surface-raised text-accent" : "text-muted hover:text-foreground"),
            )}
          >
            {item.label}
          </Link>
        </li>
      );
    });

  return (
    <>
      <nav aria-label="Navigation principale" className="hidden self-end lg:block">
        <ul className="flex items-end gap-1">{navLinks("desktop")}</ul>
      </nav>

      <div className="hidden shrink-0 items-center gap-2 lg:flex">
        <AccountActions
          user={user}
          variant="desktop"
          isSpacePage={isSpacePage}
          onNavigate={close}
        />
      </div>

      <button
        ref={toggleRef}
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        aria-controls={menuId}
        aria-label={open ? "Fermer le menu" : "Ouvrir le menu"}
        className="-mr-2 inline-flex size-11 items-center justify-center border-2 border-edge text-foreground hover:bg-surface-raised lg:hidden"
      >
        <MenuIcon open={open} />
      </button>

      <div
        id={menuId}
        hidden={!open}
        className="absolute inset-x-0 top-full border-b-4 border-edge bg-background px-4 pt-2 pb-6 lg:hidden"
      >
        <nav aria-label="Navigation principale">
          <ul className="flex flex-col gap-1">{navLinks("mobile")}</ul>
        </nav>
        {user !== undefined && (
          <div className="mt-4 grid grid-cols-2 items-center gap-3 border-t border-border pt-4">
            <AccountActions
              user={user}
              variant="mobile"
              isSpacePage={isSpacePage}
              onNavigate={close}
            />
          </div>
        )}
      </div>
    </>
  );
}
