# Contrat avec le backend

Le contrat de l'API Go est [backend/openapi/openapi.yaml](../../backend/openapi/openapi.yaml)
(« spec d'abord » : les tests du backend échouent si une route diverge du fichier). Ce document
dit **où le frontend actuel s'écarte de ce contrat** et ce qu'il faut changer au moment du
branchement. À mettre à jour à chaque fonction branchée.

> État au 20/10/2026 : **inscription, activation, connexion, session, déconnexion et
> réinitialisation du mot de passe sont branchées** sur l'API dès que `API_URL` est renseignée.
> Tout le reste fonctionne encore sur des mocks, y compris avec l'API.

## 1. Couverture : ce qui est dans le contrat, et le reste

Le contrat décrit aujourd'hui **40 opérations** : identité et modération des comptes, mais aussi
incidents, interventions, types d'incident et super-vilains (plus `/health` et `/ready`). Les
autres domaines du frontend n'ont pas encore de contrat.

| Domaine frontend                   | Fonctions `lib/api/`                           | Contrat OpenAPI                                    | État       |
| ---------------------------------- | ---------------------------------------------- | -------------------------------------------------- | ---------- |
| Connexion, session, déconnexion    | `login`, `getSessionUser`, `logout`            | `/auth/login`, `/auth/me`, `/auth/logout`          | ✅ branché |
| Inscription                        | `register`                                     | `/auth/register`                                   | ✅ branché |
| Activation du compte               | `activateAccount`, `resendActivation`          | `/auth/activate`, `/auth/resend-activation`        | ✅ branché |
| Mot de passe oublié (demande)      | `requestPasswordReset`                         | `/auth/password-reset`                             | ✅ branché |
| Nouveau mot de passe (jeton)       | `confirmPasswordReset`                         | `/auth/password-reset/confirm`                     | ✅ branché |
| Demande de statut super-héros      | `getHeroApplications`, `reviewHeroApplication` | `/heroes/requests`, `/admin/hero-requests/*`       | mock       |
| Comptes (suspension, vérification) | `getCitizenAccounts`, `moderateAccount`        | `/admin/users/*`                                   | mock       |
| Incidents, interventions           | `lib/api/incidents.ts`                         | `/incidents/*`, `/interventions/*` (§ 3)           | mock       |
| Types d'incident                   | `IncidentType` (`types/incident.ts`), en dur   | `/intervention-types`, `/admin/intervention-types` | en dur     |
| Notes et avis                      | `ratings.ts`                                   | ❌ aucun                                           | mock       |
| Super-vilains                      | `villains.ts`                                  | `/villains` (liste, création, modification)        | mock       |
| Fiches super-héros                 | `super-heroes.ts` (profils)                    | ❌ aucun                                           | mock       |
| Vidéothèque, blog, newsletter      | `videos.ts`, `blog.ts`, `newsletter.ts`        | ❌ aucun                                           | mock       |
| Formulaire de contact              | `sendContactMessage` (`contact.ts`)            | ❌ aucun                                           | mock       |
| Profil (email, téléphone, avatar)  | `account.ts`, `super-heroes.ts` (`updateMy…`)  | ❌ aucun                                           | mock       |
| Tableau de bord héros, modération  | `hero-dashboard.ts`, `moderation.ts`           | Partiel : `/admin/interventions/{id}/decision`     | mock       |

Les « Future implémentation » écrites dans les commentaires de `lib/api/` pour les domaines non
couverts sont des **hypothèses** du frontend, pas un contrat. Il faudra les faire valider (ou les
remplacer) quand le backend ajoutera ces routes.

## 2. Ce qui est branché : la session

### Principe : le serveur Next sert d'intermédiaire

Le navigateur ne parle **jamais** directement à l'API : tous les appels partent du serveur Next
(Server Actions, Server Components, `proxy.ts`), à travers le client
[src/lib/api/http.ts](../src/lib/api/http.ts) (`apiFetch`).

```
Navigateur ──cookie session──► Next (serveur) ──Cookie: session=… ──► API Go
                                  ▲                      │
                                  └──── Set-Cookie ◄─────┘  (login : recopié vers le navigateur)
```

|               | Fonctionnement                                                                                                                                 |
| ------------- | ---------------------------------------------------------------------------------------------------------------------------------------------- |
| Connexion     | `loginAction` → `login()` → `POST /auth/login`. Le jeton du `Set-Cookie` de l'API est recopié dans le cookie du navigateur (même nom)          |
| Durée         | Celle de la session API : le cookie expire à `expiresAt` (7 j max pour un citoyen ou un héros, 8 h pour un modérateur)                         |
| Lecture       | `getSessionUser(token)` → `GET /auth/me`, le jeton dans l'en-tête `Cookie`. Mémorisé le temps d'un rendu (`cache` de React)                    |
| Déconnexion   | `logoutAction` → `logout()` : `GET /auth/me` (jeton CSRF) puis `POST /auth/logout` avec `X-CSRF-Token`. Le cookie est effacé dans tous les cas |
| Nom du cookie | [cookies.ts](../src/lib/cookies.ts) : `__Host-session` en production, `session` sinon : même règle que l'API (`HEROS_COOKIE_SECURE`)           |
| API en panne  | `getSessionUser` renvoie `null` (visiteur, espaces protégés fermés) et journalise l'erreur ; `login` renvoie `unavailable`                     |
| Adresse IP    | `login` transmet `X-Forwarded-For` et `User-Agent` du visiteur : l'API limite les tentatives par IP et enregistre l'appareil de la session     |

À savoir :

- **Le nom du cookie doit être le même des deux côtés.** Si l'API pose un autre nom (par exemple
  un front lancé avec `next start` en local, donc en mode production, face à une API de
  développement), `login` lève une erreur explicite au lieu d'échouer en silence.
- **`X-Forwarded-For` n'est fiable que derrière le reverse proxy** : Caddy remplace l'en-tête
  envoyé par le client, et l'API ne croit que les adresses fixes du proxy et du front
  (`HEROS_TRUSTED_PROXIES`). Le conteneur `web` ne doit donc jamais être exposé directement.
  Sans proxy (`npm run dev`), l'API ne voit que l'adresse du serveur Next.
- Les appels du serveur Next n'envoient pas d'en-tête `Origin` : l'API les accepte sans
  `HEROS_ALLOWED_ORIGINS`. Cette liste ne sert qu'aux appels directs du navigateur.
- `proxy.ts` appelle `getUserFromToken`, donc `GET /auth/me`, sur chaque page des espaces
  protégés (voir son `matcher`), en plus des vérifications de `requireRole()` pendant le rendu.
- `rememberMe` (connexion prolongée, 30 j) existe côté API, pas encore dans `LoginCredentials`
  ni dans le formulaire.

### Domaines encore mockés avec une session réelle

Les fonctions mockées reçoivent le vrai jeton et vérifient le rôle avec la vraie session
(`userFromToken`). Mais leurs données sont indexées par les identifiants des comptes mock
(`user-1`, `hero-1`, `admin-1`) : `asMockDataUser`
([session-user.ts](../src/lib/api/session-user.ts)) donne au vrai compte l'identifiant du compte
mock de son rôle, en gardant son nom. `requireRole()` l'applique aussi, pour les composants qui
comparent `user.id` aux données (incidents d'un héros). **À supprimer quand tous les domaines
seront branchés.**

Conséquence : deux citoyens réels voient le même profil mock (celui de Camille).

### Rôles : deux vocabulaires

| Frontend (`UserRole`) | API (`UserView.role`) |
| --------------------- | --------------------- |
| `citoyen`             | `citizen`             |
| `super-heros`         | `hero`                |
| `admin`               | `moderator`           |

La traduction se fait **à un seul endroit**, `fromApiUser` dans
[http.ts](../src/lib/api/http.ts) : les composants ne voient jamais `hero` ou `moderator`.

### Erreurs de connexion

| Code API (statut)                           | `LoginError`            | Message affiché                                     |
| ------------------------------------------- | ----------------------- | --------------------------------------------------- |
| `invalid_credentials` (401)                 | `invalid-credentials`   | Identifiant ou mot de passe incorrect               |
| `validation_failed` (422)                   | `invalid-credentials`   | Idem : le formulaire ne dit pas quel champ est faux |
| `account_deactivated` (403)                 | `account-suspended`     | Compte suspendu par la modération                   |
| `account_not_activated` (403)               | `account-not-activated` | Consulter le mail d'activation                      |
| `account_locked`, `rate_limited` (429)      | `too-many-attempts`     | Réessayer dans N minutes (`Retry-After`)            |
| Autre code, API injoignable ou délai (10 s) | `unavailable`           | Connexion momentanément indisponible                |

## 3. Écarts à résoudre avant de brancher le reste

### Erreurs : enveloppe `{ error: { code, message, fields } }`

- `apiFetch` lève `ApiError` (`status`, `code`, `retryAfterSeconds`) pour toute réponse non 2xx.
  Chaque fonction mappe les codes vers son `{ ok: false, error }` typé ; un code inconnu tombe sur
  un message générique, jamais sur le `message` brut de l'API.
- Codes de champ de validation (422) : `required`, `invalid`, `too_short`, `too_long`, `weak`,
  dans `ApiError.fields` (champ imbriqué : `address.city`). Exemple de table _code → message_ :
  `toSignupErrors` dans [actions/signup.ts](../src/app/actions/signup.ts).
- Statuts à traiter explicitement : `401` (session expirée → rediriger vers `/connexion`),
  `403` (origine/CSRF/rôle), `429` (`Retry-After`), `413` et `415`.
- Écritures authentifiées : envoyer le `csrfToken` de `GET /auth/me` dans `X-CSRF-Token`
  (option `csrfToken` d'`apiFetch`, voir `logout`).

### Inscription et activation de compte

| Sujet                     | Frontend actuel                                                             | Contrat                                                                                      |
| ------------------------- | --------------------------------------------------------------------------- | -------------------------------------------------------------------------------------------- |
| Résultat de l'inscription | `SignupResult` : `{ ok, user }`, écran « vérifiez vos mails », sans session | `{ user, activationRequired: true }` : **pas de session**, un e-mail d'activation est envoyé |
| Activation                | Page `/activation?token=…` : bouton de confirmation, renvoi du lien         | Jeton reçu par e-mail à confirmer (`HEROS_ACTIVATION_PATH` = `routes.activation`)            |
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

### Incidents, interventions et super-vilains : modèles différents

Le contrat existe désormais, mais les « Future implémentation » de `lib/api/` ont été écrites
avant lui et ne lui correspondent pas. Il faudra **réécrire les fonctions d'après `openapi.yaml`**,
pas seulement changer les URL.

| Sujet            | Frontend actuel (hypothèses)                                                             | Contrat                                                                                                                                          |
| ---------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Espaces          | Routes par rôle : `/hero/incidents…`, `/me/incidents`, `/admin/incidents…`               | Routes communes à tous les rôles (`/incidents`, `/interventions/{id}`) ; « mes incidents » = `GET /incidents?reporterId=…`                       |
| Intervention     | `POST` / `PUT /hero/incidents/{id}/intervention` : un héros se positionne sur l'incident | Ressource à part : `POST /incidents/{incidentId}/interventions`, puis `join`, invitations (`confirm` / `decline`), `outcome`, `outcome/validate` |
| Types d'incident | Énumération `IncidentType` en dur                                                        | Liste lue par `GET /intervention-types`, gérée par les modérateurs                                                                               |
| Carte, priorité  | `GET /incidents/map`, `GET /hero/incidents/map`, `PUT /hero/incidents/{id}/priority`     | Absents du contrat                                                                                                                               |
| Modération       | `PUT /admin/incidents/{id}/moderation`, `PATCH /admin/incidents/{id}`                    | Absente du contrat (seul l'arbitrage du résultat d'une intervention existe)                                                                      |
| Super-vilains    | Lecture seule : `GET /villains`, `/villains/{id}`, `/villains/{id}/incidents`            | `GET /villains` (recherche `q`), `POST /villains`, `PATCH /villains/{id}` ; ni détail ni incidents d'un vilain                                   |

À trancher avec le backend avant de brancher : ce qui manque au contrat (carte, priorité,
modération des incidents, fiche d'un vilain) est-il à ajouter, ou les écrans du frontend
doivent-ils changer ?

### Pagination

Les listes du contrat utilisent `limit` (1–100, défaut 20) et `offset`, et renvoient une page
(`HeroApplicantPage`, `UserSummaryPage`). Les fonctions `getXxx` du frontend renvoient
aujourd'hui des tableaux complets : les écrans d'administration devront gérer la pagination.

### Variable d'environnement

`API_URL` (lue par le serveur au démarrage, jamais exposée au navigateur) doit pointer vers la
base **avec** le préfixe du contrat (champ `servers` de l'OpenAPI) :

| Contexte                                    | `API_URL`                                                           |
| ------------------------------------------- | ------------------------------------------------------------------- |
| Pile Docker (`make up-dev`, `make up-prod`) | `http://api:8080/api/v1` (fixé dans `backend/docker-compose.*.yml`) |
| `npm run dev` + API de `make up-dev`        | `http://localhost:8081/api/v1`                                      |
| `npm run dev` + API lancée avec `go run`    | `http://localhost:8080/api/v1`                                      |

Voir le [README](../README.md#variables-denvironnement).

## 4. Garder types et contrat synchronisés

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

Modèle à suivre : [src/lib/api/auth.ts](../src/lib/api/auth.ts) (`login`, `getSessionUser`,
`logout`).

1. La route existe-t-elle dans `openapi.yaml` ? Sinon, la faire ajouter côté backend d'abord.
2. Garder la signature ; brancher sur `isApiEnabled` : `apiFetch(…)` si l'API est configurée,
   le mock sinon (l'application doit toujours démarrer sans backend).
3. Passer le jeton dans `sessionToken`, et le `csrfToken` de `GET /auth/me` pour une écriture.
4. Convertir la réponse (rôles, statuts, dates) vers les types de `src/types/`, avec un
   commentaire `/** Schéma OpenAPI : … */` sur l'interface de la réponse brute.
5. Mapper les codes d'erreur (`ApiError.code`) vers le résultat `{ ok: false, error }` existant.
6. Mettre à jour le tableau de la section 1. Quand le domaine n'utilise plus `userFromToken`,
   vérifier si `asMockDataUser` sert encore.
