"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import { pauseHeroLocationAction, saveHeroLocationAction } from "@/app/actions/hero";

/*
 * Partage de la position d'un super-héros. Le suivi est monté dans le layout de l'espace héros :
 * la position reste à jour sur toutes ses pages, et seulement tant que l'une d'elles est ouverte
 * (un site ne lit pas la position en arrière-plan). Le panneau de « Mon profil » lit son état ici.
 */

/** Intervalle d'envoi : bien en deçà des 30 minutes après lesquelles l'API purge une position. */
const REFRESH_INTERVAL_MS = 5 * 60_000;
const PAUSE_STORAGE_KEY = "heros-sos:location-paused";

export type HeroLocationStatus = "off" | "locating" | "shared" | "denied" | "unsupported" | "error";

interface HeroLocationState {
  status: HeroLocationStatus;
  /** Message de la dernière erreur d'envoi à l'API. */
  error: string | null;
  /** « Arrêter pour l'instant » : rien n'est envoyé jusqu'à « Reprendre » (le temps de l'onglet). */
  paused: boolean;
  pause: () => Promise<void>;
  resume: () => void;
}

interface HeroLocationInternals extends HeroLocationState {
  setStatus: (status: HeroLocationStatus) => void;
  setError: (error: string | null) => void;
}

const HeroLocationContext = createContext<HeroLocationInternals | null>(null);

function readPaused(): boolean {
  try {
    return sessionStorage.getItem(PAUSE_STORAGE_KEY) === "1";
  } catch {
    return false;
  }
}

/** Sans sessionStorage (navigation privée), la pause est gardée en mémoire, le temps de la page. */
let pausedFallback = false;
const pauseListeners = new Set<() => void>();

function writePaused(paused: boolean) {
  pausedFallback = paused;
  try {
    if (paused) sessionStorage.setItem(PAUSE_STORAGE_KEY, "1");
    else sessionStorage.removeItem(PAUSE_STORAGE_KEY);
  } catch {
    // Stockage indisponible : la pause ne survit pas au rechargement.
  }
  for (const listener of pauseListeners) listener();
}

function subscribePaused(listener: () => void) {
  pauseListeners.add(listener);
  return () => pauseListeners.delete(listener);
}

const getPaused = () => readPaused() || pausedFallback;
// Le serveur n'a pas accès au sessionStorage : la pause n'est lue qu'après l'hydratation.
const getServerPaused = () => false;

export function HeroLocationProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<HeroLocationStatus>("off");
  const [error, setError] = useState<string | null>(null);
  const paused = useSyncExternalStore(subscribePaused, getPaused, getServerPaused);

  const pause = useCallback(async () => {
    writePaused(true);
    const result = await pauseHeroLocationAction();
    setError(result.ok ? null : result.error);
  }, []);

  const resume = useCallback(() => {
    writePaused(false);
    setError(null);
  }, []);

  const value = useMemo(
    () => ({ status, error, paused, pause, resume, setStatus, setError }),
    [status, error, paused, pause, resume],
  );
  return <HeroLocationContext value={value}>{children}</HeroLocationContext>;
}

function useHeroLocationInternals(): HeroLocationInternals {
  const context = useContext(HeroLocationContext);
  if (!context) throw new Error("useHeroLocation : à utiliser sous <HeroLocationProvider>");
  return context;
}

/** État du partage de position, pour l'afficher (panneau de « Mon profil »). */
export function useHeroLocation(): HeroLocationState {
  return useHeroLocationInternals();
}

/**
 * Envoie la position du navigateur tant que le héros a consenti et n'a pas mis le partage en
 * pause : tout de suite, toutes les 5 minutes, et au retour sur l'onglet. `watchPosition` ne
 * conviendrait pas : il ne se déclenche pas quand le héros ne bouge pas, et sa position serait
 * alors purgée par l'API.
 */
export function HeroLocationTracker({ enabled }: { enabled: boolean }) {
  const { paused, setStatus, setError } = useHeroLocationInternals();

  useEffect(() => {
    if (!enabled || paused) {
      setStatus("off");
      return;
    }
    if (!("geolocation" in navigator)) {
      setStatus("unsupported");
      return;
    }

    let active = true;

    // Appelée de façon asynchrone (refus du navigateur), donc après la création de `timer`.
    function stop() {
      clearInterval(timer);
      document.removeEventListener("visibilitychange", onVisible);
    }

    const locate = () => {
      navigator.geolocation.getCurrentPosition(
        async ({ coords }) => {
          const result = await saveHeroLocationAction({
            latitude: coords.latitude,
            longitude: coords.longitude,
            accuracyM: Math.round(coords.accuracy),
          });
          if (!active) return;
          setError(result.ok ? null : result.error);
          setStatus(result.ok ? "shared" : "error");
        },
        (positionError) => {
          if (!active) return;
          if (positionError.code === positionError.PERMISSION_DENIED) {
            setStatus("denied");
            stop();
          } else {
            setStatus("error");
          }
        },
        { enableHighAccuracy: false, maximumAge: 60_000, timeout: 20_000 },
      );
    };

    function onVisible() {
      if (document.visibilityState === "visible") locate();
    }

    setStatus("locating");
    locate();
    const timer = setInterval(locate, REFRESH_INTERVAL_MS);
    document.addEventListener("visibilitychange", onVisible);

    return () => {
      active = false;
      stop();
    };
  }, [enabled, paused, setStatus, setError]);

  return null;
}
