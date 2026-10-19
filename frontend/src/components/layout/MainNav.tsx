"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useEffect, useId, useRef, useState } from "react";
import { buttonClasses } from "@/components/ui/ButtonLink";
import { cn } from "@/components/ui/cn";
import { mainNav, routes } from "@/lib/routes";

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

/** Navigation principale : liens en ligne sur desktop, menu burger sur mobile. */
export function MainNav() {
  const pathname = usePathname();
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
              "block rounded-md font-medium transition-colors motion-reduce:transition-none",
              variant === "desktop" ? "px-3 py-2 text-sm" : "px-3 py-3 text-base",
              isCurrent ? "text-foreground" : "text-muted hover:text-foreground",
              isCurrent &&
                variant === "desktop" &&
                "underline decoration-primary decoration-2 underline-offset-8",
              isCurrent && variant === "mobile" && "bg-surface-raised",
            )}
          >
            {item.label}
          </Link>
        </li>
      );
    });

  return (
    <>
      <nav aria-label="Navigation principale" className="hidden md:block">
        <ul className="flex items-center gap-1">{navLinks("desktop")}</ul>
      </nav>

      <div className="hidden items-center gap-2 md:flex">
        <Link href={routes.login} className={buttonClasses("ghost")}>
          Connexion
        </Link>
        <Link href={routes.signup} className={buttonClasses("primary")}>
          Inscription
        </Link>
      </div>

      <button
        ref={toggleRef}
        type="button"
        onClick={() => setOpen((value) => !value)}
        aria-expanded={open}
        aria-controls={menuId}
        aria-label={open ? "Fermer le menu" : "Ouvrir le menu"}
        className="-mr-2 inline-flex size-11 items-center justify-center rounded-md text-foreground hover:bg-surface-raised md:hidden"
      >
        <MenuIcon open={open} />
      </button>

      <div
        id={menuId}
        hidden={!open}
        className="absolute inset-x-0 top-full border-b border-border bg-background px-4 pt-2 pb-6 shadow-2xl md:hidden"
      >
        <nav aria-label="Navigation principale">
          <ul className="flex flex-col gap-1">{navLinks("mobile")}</ul>
        </nav>
        <div className="mt-4 grid grid-cols-2 gap-3 border-t border-border pt-4">
          <Link href={routes.login} onClick={close} className={buttonClasses("secondary")}>
            Connexion
          </Link>
          <Link href={routes.signup} onClick={close} className={buttonClasses("primary")}>
            Inscription
          </Link>
        </div>
      </div>
    </>
  );
}
