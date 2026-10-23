"use client";

import { useCallback } from "react";
import type { BuiltCommand } from "@/lib/offline/forms";
import type { FormState } from "@/lib/validation/form-state";
import { useOffline } from "./OfflineProvider";

/** Un échec de `fetch` : le réseau manque. Navigateurs : TypeError (« Failed to fetch », « Load failed »…). */
export const isNetworkFailure = (error: unknown) => error instanceof TypeError;

/**
 * Enveloppe une Server Action de formulaire pour qu'elle survive à une coupure de réseau : hors ligne, ou si
 * l'envoi échoue faute de réseau, l'action est rangée sur l'appareil (voir OfflineProvider) au lieu d'échouer,
 * et le formulaire affiche une confirmation. Sans file disponible (API absente, utilisateur inconnu), l'action
 * s'exécute telle quelle, comme avant.
 */
export function useOfflineAction<State extends FormState<string>>(
  action: (previous: State, formData: FormData) => Promise<State>,
  build: (formData: FormData) => BuiltCommand<State>,
): (previous: State, formData: FormData) => Promise<State> {
  const { enabled, enqueue } = useOffline();

  return useCallback(
    async (previous, formData) => {
      const queue = async (): Promise<State> => {
        const built = build(formData);
        if (!built.ok) return built.state;
        if (!(await enqueue(built.draft))) {
          return {
            status: "error",
            message: "Impossible de garder cette action sur l'appareil. Réessayez avec du réseau.",
            fieldErrors: {},
          } as State;
        }
        return {
          status: "success",
          message: `${built.message} Elle sera envoyée dès le retour du réseau.`,
          fieldErrors: {},
        } as State;
      };

      if (enabled && !navigator.onLine) return queue();
      try {
        return await action(previous, formData);
      } catch (error) {
        if (enabled && isNetworkFailure(error)) return queue();
        throw error;
      }
    },
    [action, build, enabled, enqueue],
  );
}
