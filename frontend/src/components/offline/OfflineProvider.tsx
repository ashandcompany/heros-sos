"use client";

import { useRouter } from "next/navigation";
import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  useSyncExternalStore,
  type ReactNode,
} from "react";
import type { CommandDraft } from "@/lib/offline/commands";
import { createOutbox, type Outbox, type OutboxEntry } from "@/lib/offline/outbox";
import { createSealer } from "@/lib/offline/seal";
import { sendCommand } from "@/lib/offline/sender";
import { createIndexedDbStores } from "@/lib/offline/store";

/*
 * File des actions faites hors ligne, à l'échelle du site : elle range une action quand le réseau manque, la
 * rejoue dès son retour, et dit à l'utilisateur où elle en est. Le serveur reste l'autorité : une action refusée
 * (un autre héros est déjà sur place…) est montrée avec sa raison, pas rejouée en boucle.
 *
 * La file n'existe que pour un utilisateur connecté, avec l'API (OfflineIdentity la renseigne depuis le serveur).
 */

interface OfflineContextValue {
  /** Vrai quand les actions peuvent être mises en file : API active, utilisateur connu, stockage disponible. */
  enabled: boolean;
  online: boolean;
  entries: OutboxEntry[];
  notice: string | null;
  /** Range une action et tente de l'envoyer ; faux si la file n'est pas disponible. */
  enqueue: (draft: CommandDraft) => Promise<boolean>;
  dismiss: (id: string) => Promise<void>;
  dismissNotice: () => void;
  identify: (userId: string | null) => void;
}

const unavailable: OfflineContextValue = {
  enabled: false,
  online: true,
  entries: [],
  notice: null,
  enqueue: async () => false,
  dismiss: async () => {},
  dismissNotice: () => {},
  identify: () => {},
};

const OfflineContext = createContext<OfflineContextValue>(unavailable);

export const useOffline = () => useContext(OfflineContext);

/** Compte dont les pages sont dans le cache du service worker : un autre compte les efface. */
const CACHE_OWNER_KEY = "heros-sos:offline-owner";
const PAGE_CACHE_PREFIX = "heros-pages-";
const SERVICE_WORKER_URL = "/sw.js";
const OUTBOX_LOCK = "heros-sos-outbox";
const BACKGROUND_SYNC_TAG = "heros-outbox";

/** Délais entre deux tentatives quand le serveur ne répond pas : de 5 s à 1 min. */
const RETRY_FIRST_MS = 5_000;
const RETRY_MAX_MS = 60_000;

const subscribeToConnection = (callback: () => void) => {
  window.addEventListener("online", callback);
  window.addEventListener("offline", callback);
  return () => {
    window.removeEventListener("online", callback);
    window.removeEventListener("offline", callback);
  };
};

/** Efface les pages gardées pour la lecture hors ligne (déconnexion, changement de compte). */
export async function clearOfflinePages(): Promise<void> {
  if (typeof caches === "undefined") return;
  const names = await caches.keys();
  await Promise.all(
    names.filter((name) => name.startsWith(PAGE_CACHE_PREFIX)).map((name) => caches.delete(name)),
  );
}

function openOutbox(): Outbox | null {
  if (typeof indexedDB === "undefined" || typeof crypto === "undefined" || !crypto.subtle)
    return null;
  const stores = createIndexedDbStores();
  return createOutbox({ store: stores.commands, sealer: createSealer(stores.keys) });
}

function plural(count: number, one: string, many: string) {
  return `${count} ${count > 1 ? many : one}`;
}

export function OfflineProvider({ children }: { children: ReactNode }) {
  const router = useRouter();
  const online = useSyncExternalStore(
    subscribeToConnection,
    () => navigator.onLine,
    () => true,
  );
  const [userId, setUserId] = useState<string | null>(null);
  const [entries, setEntries] = useState<OutboxEntry[]>([]);
  const [notice, setNotice] = useState<string | null>(null);
  const [enabled, setEnabled] = useState(false);
  const outbox = useRef<Outbox | null>(null);
  const retry = useRef<{ timer?: ReturnType<typeof setTimeout>; delay: number }>({
    delay: RETRY_FIRST_MS,
  });
  const flushLatest = useRef<() => Promise<void>>(async () => {});

  const refresh = useCallback(async () => {
    if (outbox.current && userId) setEntries(await outbox.current.list(userId));
  }, [userId]);

  const flush = useCallback(async () => {
    const current = outbox.current;
    if (!current || !userId || !navigator.onLine) return;
    // Un seul onglet envoie à la fois : les autres laissent faire (Web Locks, quand le navigateur les a).
    const run = () => current.flush(userId, sendCommand);
    const report = navigator.locks
      ? await navigator.locks.request(OUTBOX_LOCK, { ifAvailable: true }, (lock) =>
          lock ? run() : null,
        )
      : await run();
    if (!report) return;
    await refresh();

    if (report.stoppedBy === "retry") {
      const { delay } = retry.current;
      retry.current.delay = Math.min(delay * 2, RETRY_MAX_MS);
      clearTimeout(retry.current.timer);
      retry.current.timer = setTimeout(() => void flushLatest.current(), delay);
      return;
    }
    retry.current.delay = RETRY_FIRST_MS;
    if (report.synced > 0) {
      setNotice(`${plural(report.synced, "action synchronisée", "actions synchronisées")}.`);
      // Les pages affichent des données du serveur : elles se rechargent pour montrer ce qui vient de partir.
      router.refresh();
    }
    if (report.rejected > 0) {
      setNotice(
        `${plural(report.rejected, "action refusée", "actions refusées")} par le serveur : voir le détail.`,
      );
    }
  }, [userId, refresh, router]);

  useEffect(() => {
    flushLatest.current = flush;
  }, [flush]);

  // Un utilisateur connu : ouverture de la file, lecture de son contenu, premier envoi.
  useEffect(() => {
    if (!userId) {
      outbox.current = null;
      return;
    }
    const opened = openOutbox();
    outbox.current = opened;
    const retryState = retry.current;
    void (async () => {
      if (!opened) return;
      setEnabled(true);
      await refresh();
      await flush();
    })();
    return () => {
      clearTimeout(retryState.timer);
      setEnabled(false);
    };
    // La file ne se rouvre que pour un autre utilisateur : refresh et flush en dépendent déjà.
  }, [userId, refresh, flush]);

  // Les pages gardées pour la lecture hors ligne sont celles d'un compte : un autre compte ne doit pas les voir.
  useEffect(() => {
    if (!userId) return;
    const previous = localStorage.getItem(CACHE_OWNER_KEY);
    if (previous !== userId) {
      localStorage.setItem(CACHE_OWNER_KEY, userId);
      if (previous) void clearOfflinePages();
    }
  }, [userId]);

  // Service worker (cache de lecture hors ligne, réveil par Background Sync) et déclencheurs d'envoi.
  useEffect(() => {
    if (!userId) return;
    const wake = () => void flushLatest.current();
    const onVisible = () => document.visibilityState === "visible" && wake();
    window.addEventListener("online", wake);
    document.addEventListener("visibilitychange", onVisible);
    const onMessage = (event: MessageEvent) => {
      if ((event.data as { type?: string } | null)?.type === "flush") wake();
    };
    navigator.serviceWorker?.addEventListener("message", onMessage);
    void navigator.serviceWorker?.register(SERVICE_WORKER_URL, { scope: "/" }).catch(() => {});
    return () => {
      window.removeEventListener("online", wake);
      document.removeEventListener("visibilitychange", onVisible);
      navigator.serviceWorker?.removeEventListener("message", onMessage);
    };
  }, [userId]);

  const enqueue = useCallback(
    async (draft: CommandDraft) => {
      if (!outbox.current || !userId) return false;
      try {
        await outbox.current.enqueue(userId, draft);
      } catch {
        return false;
      }
      await refresh();
      // Demande à ce que le navigateur réveille le service worker au retour du réseau, même onglet fermé.
      void navigator.serviceWorker?.ready
        .then((registration) =>
          (
            registration as ServiceWorkerRegistration & {
              sync?: { register(tag: string): Promise<void> };
            }
          ).sync?.register(BACKGROUND_SYNC_TAG),
        )
        .catch(() => {});
      void flushLatest.current();
      return true;
    },
    [userId, refresh],
  );

  const dismiss = useCallback(
    async (id: string) => {
      await outbox.current?.dismiss(id);
      await refresh();
    },
    [refresh],
  );

  const value = useMemo<OfflineContextValue>(
    () => ({
      enabled,
      online,
      entries,
      notice,
      enqueue,
      dismiss,
      dismissNotice: () => setNotice(null),
      identify: setUserId,
    }),
    [enabled, online, entries, notice, enqueue, dismiss],
  );

  return <OfflineContext.Provider value={value}>{children}</OfflineContext.Provider>;
}

/**
 * Renseigne l'utilisateur connecté (lu côté serveur : le navigateur ne le connaît pas). `null` : visiteur, ou
 * API absente, donc pas de file.
 */
export function OfflineIdentity({ userId }: { userId: string | null }) {
  const { identify } = useOffline();
  useEffect(() => {
    identify(userId);
    return () => identify(null);
  }, [identify, userId]);
  return null;
}
