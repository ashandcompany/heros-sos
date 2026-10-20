import { getMyAccount } from "./account";
import { getSessionUser } from "./auth";
import { getMyHeroProfile } from "./super-heroes";

/**
 * Photo de profil de l'utilisateur connecté, pour la barre de navigation (`null` : initiales).
 * Ne lève jamais d'erreur : une photo introuvable ne doit pas empêcher d'afficher la navigation.
 * Future implémentation : un champ `avatarUrl` dans la réponse de `GET ${API_URL}/auth/me`, pour
 * éviter cet appel supplémentaire.
 */
export async function getMyAvatarUrl(token: string): Promise<string | null> {
  try {
    const user = await getSessionUser(token);
    if (user?.role === "citoyen") return (await getMyAccount(token)).avatarUrl;
    if (user?.role === "super-heros") return (await getMyHeroProfile(token)).avatarUrl;
    return null;
  } catch {
    return null;
  }
}
