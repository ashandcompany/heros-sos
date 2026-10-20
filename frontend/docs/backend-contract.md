# Contrat avec le backend

Le contrat de l'API Go est [backend/openapi/openapi.yaml](../../backend/openapi/openapi.yaml)
(« spec d'abord » : les tests du backend échouent si une route diverge du fichier). Ce document
dit **où le frontend actuel s'écarte de ce contrat** et ce qu'il faut changer au moment du
branchement. À mettre à jour à chaque fonction branchée.

> État au 20/10/2026 : le frontend fonctionne entièrement sur des mocks. Aucune fonction de
> `src/lib/api/` n'appelle le backend.

## 1. Couverture : ce qui est dans le contrat, et le reste

Le contrat décrit aujourd'hui **20 opérations** (identité et modération des comptes). Tout le reste
du frontend n'a pas encore de contrat.

| Domaine frontend                   | Fonctions `lib/api/`                           | Contrat OpenAPI                                               |
| ---------------------------------- | ---------------------------------------------- | ------------------------------------------------------------- |
| Connexion, session, déconnexion    | `login`, `getSessionUser` + `logoutAction`     | `/auth/login`, `/auth/me`, `/auth/logout`                     |
| Inscription                        | `register`                                     | `/auth/register`, `/auth/activate`, `/auth/resend-activation` |
| Mot de passe oublié                | `requestPasswordReset`                         | `/auth/password-reset` (+ `/confirm`)                         |
| Demande de statut super-héros      | `getHeroApplications`, `reviewHeroApplication` | `/heroes/requests`, `/admin/hero-requests/*`                  |
| Comptes (suspension, vérification) | `getCitizenAccounts`, `moderateAccount`        | `/admin/users/*`                                              |
| Incidents, interventions, notes    | `lib/api/incidents.ts`, `ratings.ts`           | ❌ aucun                                                      |
| Fiches super-héros / super-vilains | `super-heroes.ts` (profils), `villains.ts`     | ❌ aucun                                                      |
| Vidéothèque, blog, newsletter      | `videos.ts`, `blog.ts`, `newsletter.ts`        | ❌ aucun                                                      |
| Profil (email, téléphone, avatar)  | `account.ts`, `super-heroes.ts` (`updateMy…`)  | ❌ aucun                                                      |
| Tableau de bord héros, modération  | `hero-dashboard.ts`, `moderation.ts`           | ❌ aucun                                                      |

Les « Future implémentation » écrites dans les commentaires de `lib/api/` pour les domaines non
couverts sont des **hypothèses** du frontend, pas un contrat. Il faudra les faire valider (ou les
remplacer) quand le backend ajoutera ces routes.

## 2. Écarts à résoudre avant de brancher

### Session : cookie posé par l'API, plus par Next

|               | Frontend actuel                                              | Contrat                                                                                               |
| ------------- | ------------------------------------------------------------ | ----------------------------------------------------------------------------------------------------- |
| Cookie        | Posé par `loginAction` ([cookies.ts](../src/lib/cookies.ts)) | Posé par l'API (`Set-Cookie` de `POST /auth/login`)                                                   |
| Nom           | `session`                                                    | `session` en dev, **`__Host-session`** en production (HTTPS)                                          |
| Valeur        | Jeton mock (`mock-token-<id>`)                               | Jeton opaque aléatoire, jamais lisible en JavaScript                                                  |
| Envoi à l'API | « `Authorization: Bearer` » (commentaires de `lib/api/`)     | Le cookie, tel quel. Pas d'en-tête `Authorization`                                                    |
| Écritures     | —                                                            | En-tête **`X-CSRF-Token`** obligatoire (POST/PUT/PATCH/DELETE)                                        |
| Durée         | 7 jours fixes                                                | 2 h d'inactivité / 7 j max (citoyen, héros) ; 15 min / 8 h (modérateur) ; « se souvenir de moi » 30 j |

À faire :

- Dans une Server Action, l'appel `fetch` vers l'API doit **recopier le `Set-Cookie`** de la
  réponse de login vers le navigateur (`cookies().set(…)`), avec le même nom, et transmettre le
  cookie de la requête entrante sur les appels suivants.
- Remplacer `SESSION_COOKIE` par un nom dépendant de l'environnement (`__Host-session` en
  production), ou le lire depuis une configuration partagée avec le backend. **`proxy.ts` et
  `server-session.ts` en dépendent.**
- Conserver le `csrfToken` renvoyé par `/auth/login` et `/auth/me` et l'envoyer dans
  `X-CSRF-Token` sur toute écriture. Il peut être relu via `GET /auth/me` (`getSessionUser`) à
  chaque rendu serveur : pas besoin de le stocker côté client.
- `proxy.ts` appelle `getUserFromToken` à **chaque requête** : une fois branchée, cela devient un
  aller-retour vers `GET /auth/me`. Prévoir un cache court ou limiter le proxy à un test de
  présence du cookie, la vraie vérification restant dans `requireRole()`.
- Le backend vérifie l'en-tête `Origin` : l'origine du site Next doit figurer dans
  `HEROS_ALLOWED_ORIGINS` (valeur d'exemple : `http://localhost:3000`).
- `rememberMe` (connexion prolongée) existe côté API, pas encore dans `LoginCredentials` ni dans
  le formulaire.

### Rôles : deux vocabulaires

| Frontend (`UserRole`) | API (`UserView.role`) |
| --------------------- | --------------------- |
| `citoyen`             | `citizen`             |
| `super-heros`         | `hero`                |
| `admin`               | `moderator`           |

Le frontend utilise ses valeurs partout (`proxy.ts`, `routes.ts`, `requireRole`…). La
traduction doit se faire **à un seul endroit**, dans `lib/api/` (un mapper `fromApiRole`), pour
que les composants ne voient jamais `hero` ou `moderator`.

### Erreurs : enveloppe `{ error: { code, message, fields } }`

- L'API renvoie des codes **stables** (`invalid_credentials`, `account_locked`,
  `account_not_activated`, `account_deactivated`…). Les `lib/api/` mappent ces codes vers les
  `error` déjà typés (`"invalid-credentials"`, `"account-suspended"`…). Un code inconnu doit
  tomber sur un message générique, jamais sur le `message` brut.
- Codes de champ de validation (422) : `required`, `invalid`, `too_short`, `too_long`, `weak`.
  Les messages français vivent aujourd'hui dans `lib/validation/` ; il faudra une table
  _code de champ → message_ pour afficher les erreurs renvoyées par l'API dans `fieldErrors`.
- Statuts à traiter explicitement : `401` (session expirée → rediriger vers `/connexion`),
  `403` (origine/CSRF/rôle), `429` (`Retry-After`), `413` et `415`.
- **Compte verrouillé** (`account_locked`, avec `Retry-After`) : état non géré par le frontend
  (`LoginResult` ne connaît que `invalid-credentials` et `account-suspended`).

### Inscription et activation de compte

| Sujet                     | Frontend actuel                                                             | Contrat                                                                                      |
| ------------------------- | --------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Résultat de l'inscription | `SignupResult` : `{ ok, user, token }` (connexion immédiate)                | `{ user, activationRequired: true }` : **pas de session**, un e-mail d'activation est envoyé |
| Activation                | **Aucune page** (`/auth/activate`, `/auth/resend-activation`)               | Jeton reçu par e-mail à confirmer                                                            |
| Mot de passe oublié       | Page de demande seulement                                                   | Il manque la page qui consomme le jeton (`/auth/password-reset/confirm`)                     |
| Adresse                   | `PostalAddress` : rue, code postal, ville, pays                             | Idem + champ `complement` facultatif                                                         |
| Mot de passe              | `lib/validation/signup.ts` : ≥ 10 caractères, majuscule, minuscule, chiffre | Mêmes règles (`weak` sinon) : déjà alignés                                                   |
| Téléphone                 | Règles frontend                                                             | Mobile FR (06/07) ou international (+33…), espaces et tirets ignorés                         |

Les règles de validation du frontend et du backend doivent rester **identiques** : toute
divergence se traduit par un formulaire qui accepte une valeur que l'API refuse. À vérifier à
chaque modification de `lib/validation/`.

### Demandes de super-héros : routes différentes

Le frontend suppose `GET /admin/hero-applications` et `POST /admin/hero-applications/{id}/approve|reject`.
Le contrat expose `GET /admin/hero-requests` et
`POST /admin/hero-requests/{userId}/approve|reject`, avec un motif obligatoire au rejet
(`RejectPayload.reason`, 1000 caractères max). Le statut est `pending | approved | rejected`
(le frontend utilise `ModerationStatus`, en français : correspondance à écrire dans le mapper).

### Pagination

Les listes du contrat utilisent `limit` (1–100, défaut 20) et `offset`, et renvoient une page
(`HeroApplicantPage`, `UserSummaryPage`). Les fonctions `getXxx` du frontend renvoient
aujourd'hui des tableaux complets : les écrans d'administration devront gérer la pagination.

### Variable d'environnement

`NEXT_PUBLIC_API_URL` doit pointer vers la base **avec** le préfixe du contrat :
`http://localhost:8080/api/v1` (champ `servers` de l'OpenAPI). Voir le
[README](../README.md#variables-denvironnement).

## 3. Garder types et contrat synchronisés

Aujourd'hui `src/types/` est écrit à la main et les formes de l'API ne sont décrites que dans
l'OpenAPI : aucun mécanisme n'empêche une divergence silencieuse.

Règle de travail en attendant mieux :

1. Un type de `src/types/` qui correspond à un schéma OpenAPI porte un commentaire
   `/** Schéma OpenAPI : LoginResponse */`.
2. Les fonctions de `lib/api/` branchées convertissent la forme de l'API vers les types du
   frontend (rôles, statuts, dates) ; **les composants ne manipulent jamais la forme brute**.
3. Une route ajoutée au backend sans contrat est refusée par ses propres tests : le frontend
   n'a donc pas à anticiper de route absente de `openapi.yaml`.

Option recommandée quand le branchement commencera : générer les types de l'API avec
[`openapi-typescript`](https://openapi-ts.dev/) dans un fichier non versionné ou versionné à
part (`src/lib/api/schema.d.ts`), via un script `npm run api:types`. Les types écrits à la main
resteraient la forme « frontend », et le mapper de chaque fonction serait alors vérifié par le
compilateur. À ne faire que pour les domaines couverts par le contrat : générer des types pour
des routes qui n'existent pas ne servirait à rien.

## Checklist de branchement d'une fonction

1. La route existe-t-elle dans `openapi.yaml` ? Sinon, la faire ajouter côté backend d'abord.
2. Remplacer `mockResponse()` par un `fetch` vers `${API_URL}…`, en gardant la signature.
3. Transmettre le cookie de session, et `X-CSRF-Token` pour une écriture.
4. Convertir la réponse (rôles, statuts, dates) vers les types de `src/types/`.
5. Mapper les codes d'erreur vers le résultat `{ ok: false, error }` existant.
6. Mettre à jour le tableau de la section 1 et supprimer le mock devenu inutile.
