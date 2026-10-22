"use client";

import { useOffline } from "./OfflineProvider";

/** Date et heure courtes, dans le fuseau de l'appareil. */
const time = new Intl.DateTimeFormat("fr-FR", { dateStyle: "short", timeStyle: "short" });

/**
 * État de la synchronisation, en bas de l'écran : hors ligne, actions en attente d'envoi, actions refusées par
 * le serveur (avec la raison). Rien ne s'affiche quand tout va bien.
 */
export function OutboxStatus() {
  const { enabled, online, entries, notice, dismiss, dismissNotice } = useOffline();
  if (!enabled) return null;

  const pending = entries.filter((entry) => entry.state === "pending");
  const rejected = entries.filter((entry) => entry.state === "rejected");
  if (online && pending.length === 0 && rejected.length === 0 && !notice) return null;

  return (
    <aside
      aria-label="Synchronisation"
      className="fixed bottom-4 left-4 z-50 w-[min(24rem,calc(100vw-2rem))] space-y-2 border-2 border-edge bg-surface-raised p-3 font-mono text-xs shadow-[4px_4px_0_var(--color-shadow)]"
    >
      {/* Annoncé aux lecteurs d'écran à chaque changement, sans prendre le focus. */}
      <div role="status" aria-live="polite" className="space-y-1">
        {!online && (
          <p className="font-bold text-mustard uppercase">
            Hors ligne : vos actions sont gardées sur cet appareil.
          </p>
        )}
        {pending.length > 0 && (
          <p>
            {pending.length} {pending.length > 1 ? "actions en attente" : "action en attente"}{" "}
            d&apos;envoi
            {online ? " : envoi en cours…" : " : elles partiront dès le retour du réseau."}
          </p>
        )}
        {notice && (
          <p className="flex items-start justify-between gap-2 text-success">
            <span>{notice}</span>
            <button
              type="button"
              onClick={dismissNotice}
              className="underline"
              aria-label="Fermer le message"
            >
              Fermer
            </button>
          </p>
        )}
      </div>

      {pending.length > 0 && (
        <ul className="space-y-1 text-muted">
          {pending.map((entry) => (
            <li key={entry.id}>
              {entry.label} · {time.format(new Date(entry.createdAt))}
            </li>
          ))}
        </ul>
      )}

      {rejected.length > 0 && (
        <ul className="space-y-2 border-t border-border pt-2">
          {rejected.map((entry) => (
            <li key={entry.id} className="space-y-1">
              <p className="text-danger">
                <strong>{entry.label}</strong> · {time.format(new Date(entry.createdAt))}
              </p>
              <p>{entry.error}</p>
              <button type="button" onClick={() => void dismiss(entry.id)} className="underline">
                Fermer
              </button>
            </li>
          ))}
        </ul>
      )}
    </aside>
  );
}
