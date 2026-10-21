import { getMyAccount } from "./account";
import { getSessionUser } from "./auth";
import { getMyHeroProfile } from "./super-heroes";

/** Ce qu'il faut pour afficher la photo de profil (voir le composant `Avatar`). */
export interface AvatarData {
  /** Nom dont dérivent les initiales et la couleur du placeholder. */
  name: string;
  src: string | null;
}

/**
 * Photo de profil de l'utilisateur connecté, pour la barre de navigation. Le nom vient du même
 * profil que la photo (prénom du citoyen, pseudo du héros), comme sur la page de profil : sinon
 * initiales et couleur du placeholder différeraient entre les deux.
 * `null` sans profil (administrateur) ou en cas d'échec : une photo introuvable ne doit pas
 * empêcher d'afficher la navigation.
 * Future implémentation : un champ `avatarUrl` dans la réponse de `GET ${API_URL}/auth/me`, pour
 * éviter cet appel supplémentaire.
 */
export async function getMyAvatar(token: string): Promise<AvatarData | null> {
  try {
    const user = await getSessionUser(token);
    if (user?.role === "citoyen") {
      const account = await getMyAccount(token);
      return { name: account.firstName, src: account.profile?.avatarUrl ?? null };
    }
    if (user?.role === "super-heros") {
      const profile = await getMyHeroProfile(token);
      return { name: profile.alias, src: profile.avatarUrl };
    }
    return null;
  } catch {
    return null;
  }
}
