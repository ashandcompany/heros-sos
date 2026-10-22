import { isApiEnabled } from "@/lib/api/config";
import { getCurrentUser } from "@/lib/server-session";
import { LiveEvents } from "./LiveEvents";
import { OfflineIdentity } from "./OfflineProvider";

/**
 * Dit au navigateur qui est connecté (la session se lit côté serveur) : il peut alors garder les actions hors
 * ligne de cet utilisateur et suivre les événements en direct. Visiteur, ou API absente (données fictives) : rien.
 * Lit les cookies : à rendre sous <Suspense>.
 */
export async function SessionIdentity() {
  const user = isApiEnabled ? await getCurrentUser() : null;
  return (
    <>
      <OfflineIdentity userId={user?.id ?? null} />
      {user && <LiveEvents />}
    </>
  );
}
