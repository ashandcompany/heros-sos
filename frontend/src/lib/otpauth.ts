/**
 * Adresse que les « Mots de passe » d'Apple (iPhone, iPad, Mac) savent ouvrir pour ajouter un code à un compte :
 * celle de la norme otpauth (voir `MfaSetup`), avec le schéma `apple-otpauth:` que documente Apple
 * (« Securing Logins with iCloud Keychain Verification Codes »). Le lien `otpauth://` standard, lui, revient à
 * l'application d'authentification que l'appareil a désignée, qui n'est en général pas Mots de passe.
 *
 * `domain` est le domaine du site : Apple s'en sert (paramètre `issuer`) pour proposer le compte enregistré pour ce
 * site, celui de la connexion, au lieu d'en faire choisir un dans toute la liste. L'étiquette garde le nom du service.
 */
export function appleOtpauthUrl(otpauthUrl: string, domain?: string): string {
  if (!otpauthUrl.startsWith("otpauth:")) return otpauthUrl;
  const url = new URL(otpauthUrl);
  if (domain) url.searchParams.set("issuer", domain);
  return `apple-${url.toString()}`;
}
