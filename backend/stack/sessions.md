# Sessions

Le cahier §2.2 impose une gestion des sessions via le formulaire de connexion : sessions côté serveur, pas de JWT. Elles se révoquent immédiatement, par exemple quand un modérateur désactive un compte (§3.1.3).

## Jeton

- 32 octets tirés de `crypto/rand`, encodés en base64url. C'est un identifiant opaque, sans donnée.
- En base, seul le hash SHA-256 du jeton est stocké. Le jeton brut n'existe que dans le cookie : une fuite de la base ne donne aucune session utilisable.
- Table `sessions` : `token_hash`, `user_id`, `created_at`, `last_seen_at`, `expires_at`, `mfa_verified_at`, `revoked_at`, IP et résumé de l'appareil `device_label` (« Chrome · macOS », calculé à partir du User-Agent dont la chaîne brute n'est pas conservée ; données personnelles, durée de conservation limitée).
- Stockage dans la base principale, pas dans Redis : un service de moins à héberger (§2.1), révocation immédiate, une lecture par requête négligeable à cette échelle.

## Cookie

`__Host-session`, `HttpOnly`, `Secure`, `SameSite=Lax`, `Path=/`, sans attribut `Domain`.

## Next.js et API Go en même origine

Un reverse proxy (Caddy ou Traefik, dans Docker) route `/api/*` vers Go et le reste vers Next.js. Pas de CORS, et le cookie fonctionne sans configuration spéciale. Le middleware de Next ne sert qu'à rediriger : l'autorisation se vérifie toujours dans l'API Go. Un endpoint `/api/me` renvoie le rôle, utilisé pour le §5.3 (redirection d'un citoyen vers son profil, d'un héros vers son espace pro).

## Du cookie au navigateur (option A : Next relaie le cookie de Go)

La session vit dans le backend Go : il la crée, la stocke, la valide et la révoque. Le front ne fait que porter le cookie, et ne décide jamais d'un droit (il redirige pour l'ergonomie, chaque endpoint Go revérifie).

- Go crée la session et renvoie `Set-Cookie: __Host-session=…` avec l'expiration. Les attributs du cookie sont définis ici, une seule fois.
- Les Server Actions de Next (connexion, inscription, validation du code 2FA) appellent Go côté serveur, lisent le `Set-Cookie` de la réponse (`getSetCookie()`) et le réappliquent au navigateur avec `cookies().set(...)`.
- Pour les appels suivants, Next relaie l'en-tête `Cookie` reçu à Go. Go lit uniquement le cookie, jamais un en-tête `Authorization`.
- Le jeton n'est jamais lisible en JavaScript (`HttpOnly`).

## Endpoints d'authentification

Préfixe `/api/v1`, noms définitifs dans `openapi.yaml` :

| Endpoint | Rôle |
|---|---|
| `POST /auth/register` | Inscription, envoie l'email d'activation |
| `POST /auth/activate` | Active le compte avec le token du lien |
| `POST /auth/login` | Connexion par email, ou par alias pour un héros. La réponse porte `mfa` : `verified`, `not_enrolled` (session complète), `verification_required` ou `enrollment_required` (session **limitée**, voir « Règles »). |
| `GET /auth/mfa` | État de la 2FA de la session et nombre de codes de secours restants (accessible à une session limitée) |
| `POST /auth/mfa/setup` | Tout compte : tire un secret TOTP à configurer dans une application d'authentification (clé et adresse `otpauth://`) ; refusé si la 2FA est déjà activée |
| `POST /auth/mfa/enable` | Tout compte : le premier code active la 2FA, rend les codes de secours (montrés une seule fois), ouvre une session complète (nouveau jeton) et ferme les autres sessions |
| `POST /auth/mfa/verify` | Valide la session avec un code TOTP ou un code de secours à usage unique ; ouvre une session complète (nouveau jeton) |
| `POST /auth/mfa/recovery-codes` | Tout compte qui a activé la 2FA, session complète : remplace les codes de secours, contre le code TOTP courant |
| `POST /auth/logout`, `POST /auth/logout-all` | Déconnexion de l'appareil, ou de tous les appareils |
| `GET /auth/me` | Utilisateur et rôle de la session (redirections du §5.3) |
| `POST /auth/password-reset`, `POST /auth/password-reset/confirm` | Demande, puis changement avec le token reçu |
| `GET /auth/sessions`, `DELETE /auth/sessions/{id}` | Liste des appareils connectés et révocation |
| `POST /heroes/requests` | Demande du statut de super-héros (formulaire « Devenir super-héros ») |

## À adapter dans le front d'Emma

Nom du cookie `__Host-session` en production, expiration reçue de l'API au lieu des 7 jours fixes, relais de l'en-tête `Cookie` à la place du `Bearer`, état « code 2FA requis » dans le résultat de la connexion, et remplacement des jetons `mock-token-<id>` (prévisibles) par le vrai jeton aléatoire.

## Durées (à ajuster)

| Profil         | Inactivité | Durée maximale                                 |
| -------------- | ---------- | ---------------------------------------------- |
| Citoyen, héros | 2 h        | 7 jours (30 jours avec « se souvenir de moi ») |
| Modérateur     | 15 min     | 8 h                                            |

## Règles

- Nouveau jeton à la connexion, à la validation de la 2FA et au changement de mot de passe (anti-fixation).
- Mot de passe changé ou réinitialisé : toutes les autres sessions de l'utilisateur sont révoquées.
- Compte désactivé par un modérateur : ses sessions sont supprimées immédiatement.
- Compte non activé (§4.3.1) : pas de session. Héros non vérifié : rôle citoyen limité jusqu'à validation.
- CSRF : vérification de l'en-tête `Origin` et jeton CSRF sur toutes les requêtes qui modifient des données.
- 2FA (TOTP) : obligatoire pour les modérateurs (toujours en production, `HEROS_MFA_MODERATOR_REQUIRED`), facultative pour les citoyens et les super-héros, qui l'activent depuis leur page de compte. Dès qu'un compte l'a activée, elle s'applique à sa connexion, quel que soit son rôle. Après le mot de passe, la session est **limitée** : `mfa_verified_at` reste NULL et seules la 2FA (`/auth/mfa*`), `/auth/me` et la déconnexion répondent, toute autre route répond `403 mfa_required` (refus par défaut : une route doit se déclarer `ProtectedPendingMFA` pour être accessible à une session limitée). Un code valide la session et en change le jeton. Même règle pour un modérateur qui n'a pas encore activé la 2FA : sa session ne sert qu'à l'activer. Une 2FA réinitialisée limite aussitôt toutes les sessions du compte.
- Un code TOTP ne sert qu'une fois (`last_used_step`) et un code faux compte comme un échec de connexion : verrouillage progressif du compte, sessions fermées au seuil, et le mot de passe seul ne remet pas le compteur à zéro. Les codes de secours sont à usage unique.
- Aucune route ne désactive ni ne réinitialise la 2FA : l'exploitation le fait avec `make resetmfa EMAIL=…` (commande `resetmfa`, journalisée), après avoir vérifié l'identité de la personne.
- Pas encore fait : l'exigence d'une 2FA **récente** pour les actions sensibles (valider un héros, supprimer un incident) : la 2FA protège aujourd'hui la session entière, pas chaque action.
- Connexion : limitation de débit, délai croissant après échecs, message d'erreur identique que le compte existe ou non.
- « Mon compte » liste les appareils connectés avec un bouton de révocation et « se déconnecter partout ».
- Une tâche supprime les sessions expirées.
