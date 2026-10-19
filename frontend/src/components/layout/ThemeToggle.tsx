"use client";

import { useEffect, useSyncExternalStore } from "react";
import { cn } from "@/components/ui/cn";
import { THEME_STORAGE_KEY, themeColors, type Theme } from "@/lib/theme";

const THEME_EVENT = "themechange";

function readTheme(): Theme {
  return document.documentElement.dataset.theme === "light" ? "light" : "dark";
}

function applyTheme(theme: Theme) {
  if (theme === "light") document.documentElement.dataset.theme = "light";
  else delete document.documentElement.dataset.theme;
}

function subscribe(onChange: () => void) {
  // Choix fait dans un autre onglet : on l'applique aussi à celui-ci.
  const onStorage = (event: StorageEvent) => {
    if (event.key !== THEME_STORAGE_KEY) return;
    applyTheme(event.newValue === "light" ? "light" : "dark");
    onChange();
  };
  window.addEventListener(THEME_EVENT, onChange);
  window.addEventListener("storage", onStorage);
  return () => {
    window.removeEventListener(THEME_EVENT, onChange);
    window.removeEventListener("storage", onStorage);
  };
}

/** Thème inconnu côté serveur : le switch est rendu neutre, puis synchronisé à l'hydratation. */
const getServerTheme = () => null;

/** Interrupteur thème sombre / thème clair, mémorisé dans le navigateur. */
export function ThemeToggle({ className }: { className?: string }) {
  const theme = useSyncExternalStore(subscribe, readTheme, getServerTheme);
  const isLight = theme === "light";

  useEffect(() => {
    if (!theme) return;
    document.querySelector('meta[name="theme-color"]')?.setAttribute("content", themeColors[theme]);
  }, [theme]);

  const toggle = () => {
    const next: Theme = isLight ? "dark" : "light";
    applyTheme(next);
    try {
      if (next === "light") localStorage.setItem(THEME_STORAGE_KEY, "light");
      else localStorage.removeItem(THEME_STORAGE_KEY);
    } catch {
      // Stockage indisponible (navigation privée…) : le thème s'applique pour cette page seulement.
    }
    window.dispatchEvent(new Event(THEME_EVENT));
  };

  return (
    <button
      type="button"
      role="switch"
      aria-checked={isLight}
      onClick={toggle}
      className={cn(
        "group inline-flex min-h-6 shrink-0 items-center gap-2 rounded-sm font-mono text-xs tracking-widest uppercase hover:text-foreground",
        className,
      )}
    >
      <span className="sm:hidden">Clair</span>
      <span className="hidden sm:inline">Mode clair</span>
      <span
        aria-hidden="true"
        className={cn(
          "relative inline-flex h-4 w-8 rounded-full border-2 border-border transition-colors motion-reduce:transition-none",
          isLight ? "bg-primary" : "bg-background",
        )}
      >
        <span
          className={cn(
            "absolute top-1/2 left-0.5 size-2 -translate-y-1/2 rounded-full bg-foreground transition-transform motion-reduce:transition-none",
            isLight && "translate-x-4",
          )}
        />
      </span>
    </button>
  );
}
