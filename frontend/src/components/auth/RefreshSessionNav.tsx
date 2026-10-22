"use client";

import { useRouter } from "next/navigation";
import { useEffect } from "react";

/**
 * Relit la session de la navbar à l'arrivée sur la page.
 *
 * La navbar est rendue dans le layout racine, que Next garde tel quel lors d'une navigation côté client :
 * quand une session expirée (inactivité d'un modérateur, révocation…) mène à la connexion, elle afficherait
 * encore l'utilisateur comme connecté. `router.refresh()` refait le rendu serveur des layouts.
 */
export function RefreshSessionNav() {
  const router = useRouter();

  useEffect(() => {
    router.refresh();
  }, [router]);

  return null;
}
