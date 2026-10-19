"use client";

import { useSyncExternalStore } from "react";

/**
 * Horloge partagée : l'instant courant n'est lu qu'à chaque tic, jamais dans getSnapshot.
 * getSnapshot doit renvoyer la même valeur entre deux notifications, sinon React boucle.
 */
let now: Date | null = null;
const listeners = new Set<() => void>();
let timer: ReturnType<typeof setInterval> | undefined;

function subscribe(onChange: () => void) {
  listeners.add(onChange);
  if (!timer) {
    now = new Date();
    timer = setInterval(() => {
      now = new Date();
      listeners.forEach((listener) => listener());
    }, 1000);
  }
  return () => {
    listeners.delete(onChange);
    if (listeners.size === 0) {
      clearInterval(timer);
      timer = undefined;
    }
  };
}

const getNow = () => now;
const getServerNow = () => null;

function useNow(): Date | null {
  return useSyncExternalStore(subscribe, getNow, getServerNow);
}

/** Horloge numérique (heure réelle). Décorative : masquée aux lecteurs d'écran. */
export function DigitalClock() {
  const date = useNow();

  return (
    <span aria-hidden="true" className="digital text-sm">
      {date ? date.toLocaleTimeString("fr-FR", { hour12: false }) : "--:--:--"}
    </span>
  );
}

/** Date du jour, pour la mention « Document généré le … » (rendue côté client). */
export function GeneratedOn() {
  const date = useNow();
  return (
    <>
      {date
        ? date.toLocaleDateString("fr-FR", { day: "2-digit", month: "long", year: "numeric" })
        : "…"}
    </>
  );
}
