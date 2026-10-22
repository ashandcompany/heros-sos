import { unstable_rethrow } from "next/navigation";
import { getMyHeroProfile } from "@/lib/api";
import { requireHeroSession } from "@/lib/server-session";
import { HeroLocationTracker } from "./HeroLocation";

/** Le héros a-t-il consenti au partage de sa position ? `null` si son profil est illisible. */
async function readLocationConsent(token: string): Promise<boolean | null> {
  try {
    return (await getMyHeroProfile(token)).locationConsentAt !== null;
  } catch (error) {
    unstable_rethrow(error);
    console.error("Suivi de la position : profil illisible", error);
    return null;
  }
}

/**
 * Lance le suivi de la position si le héros y a consenti (`GET /me`, déjà mémorisé pour la page).
 * Lit la session : à rendre sous <Suspense>. Un profil illisible n'empêche pas d'afficher la page :
 * le suivi reste simplement arrêté.
 */
export async function HeroLocationTrackerLoader() {
  const { token } = await requireHeroSession();
  const consent = await readLocationConsent(token);
  return consent === null ? null : <HeroLocationTracker enabled={consent} />;
}
