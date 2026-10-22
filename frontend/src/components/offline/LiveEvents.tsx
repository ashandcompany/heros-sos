"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

/** Chemin du flux temps réel de l'API (SSE), servi sur la même origine par le reverse proxy. */
const STREAM_URL = "/api/v1/events/stream";

/** Plusieurs événements d'affilée (une intervention en compte une dizaine) ne rechargent la page qu'une fois. */
const REFRESH_DELAY_MS = 400;

/**
 * Flux temps réel : à chaque événement du journal (incident déclaré ou modéré, intervention démarrée, résultat…), la page
 * se recharge avec les données à jour. `EventSource` se reconnecte seul après une coupure en renvoyant le numéro
 * du dernier événement reçu : le serveur rejoue ce qui a été manqué, rien n'est perdu.
 *
 * Le flux passe par le reverse proxy (`/api/v1/*` va à l'API) : sans lui (front lancé seul sur le port 3000), la
 * connexion échoue sans bruit et la page reste simplement non rafraîchie.
 */
export function LiveEvents() {
  const router = useRouter();

  useEffect(() => {
    const source = new EventSource(STREAM_URL, { withCredentials: true });
    let pending: ReturnType<typeof setTimeout> | undefined;
    const onEvent = () => {
      clearTimeout(pending);
      pending = setTimeout(() => router.refresh(), REFRESH_DELAY_MS);
    };
    // Un type d'événement par nom (event: incident.declared…) : on les écoute tous, sans les énumérer.
    source.onmessage = onEvent;
    for (const type of eventTypes) source.addEventListener(type, onEvent);
    return () => {
      clearTimeout(pending);
      source.close();
    };
  }, [router]);

  return null;
}

const eventTypes = [
  "incident.declared",
  "incident.moderated",
  "incident.corrected",
  "incident.priority_changed",
  "intervention.started",
  "intervention.joined",
  "intervention.invited",
  "intervention.invitation_confirmed",
  "intervention.invitation_declined",
  "intervention.outcome_proposed",
  "intervention.outcome_validated",
  "intervention.outcome_disputed",
  "intervention.closed",
  "intervention.arbitrated",
];
