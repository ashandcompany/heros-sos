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

| Domaine frontend                      | Fonctions `lib/api/`                                                                                         | Contrat OpenAPI                                                            | État       |
| ------------------------------------- | ------------------------------------------------------------------------------------------------------------ | -------------------------------------------------------------------------- | ---------- |
| Connexion, session, déconnexion       | `login`, `getSessionUser`, `logout`                                                                          | `/auth/login`, `/auth/me`, `/auth/logout`                                  | ✅ branché |
| Inscription                           | `register`                                                                                                   | `/auth/register`                                                           | ✅ branché |
| Activation du compte                  | `activateAccount`, `resendActivation`                                                                        | `/auth/activate`, `/auth/resend-activation`                                | ✅ branché |
| Mot de passe oublié (demande)         | `requestPasswordReset`                                                                                       | `/auth/password-reset`                                                     | ✅ branché |
| Nouveau mot de passe (jeton)          | `confirmPasswordReset`                                                                                       | `/auth/password-reset/confirm`                                             | ✅ branché |
| Demande de statut super-héros         | `hero-requests.ts` (`requestHeroStatus`, `getMyHeroRequest`, `getHeroApplications`, `reviewHeroApplication`) | `/heroes/requests`, `/heroes/requests/me`, `/admin/hero-requests/*`        | ✅ branché |
| Comptes (désactivation, vérification) | `user-moderation.ts` (`getAccounts`, `moderateAccount`)                                                      | `/admin/users`, `/admin/users/{id}/deactivate`, `…/reactivate`, `…/verify` | ✅ branché |
| Incidents, interventions              | `lib/api/incidents.ts`                                                                                       | `/incidents/*`, `/interventions/*` (§ 3)                                   | mock       |
| Types d'incident                      | `incident-types.ts` (`getIncidentTypes`, `getAllIncidentTypes`, `createIncidentType`)                        | `/intervention-types`, `/admin/intervention-types`                         | ✅ branché |
| Notes et avis                         | `ratings.ts`                                                                                                 | ❌ aucun                                                                   | mock       |
| Super-vilains                         | `villains.ts`                                                                                                | `/villains` (liste, création, modification)                                | mock       |
| Fiches super-héros                    | `super-heroes.ts` (profils)                                                                                  | ❌ aucun                                                                   | mock       |
| Vidéothèque, blog, newsletter         | `videos.ts`, `blog.ts`, `newsletter.ts`                                                                      | ❌ aucun                                                                   | mock       |
| Formulaire de contact                 | `sendContactMessage` (`contact.ts`)                                                                          | ❌ aucun                                                                   | mock       |
| Profil, changements d'email           | `account.ts`, `super-heroes.ts` (`updateMy…`)                                                                | partiel : `GET /auth/me` (prénom, email)                                   | partiel    |
| Tableau de bord héros, modération     | `hero-dashboard.ts`, `moderation.ts`                                                                         | Partiel : `/admin/interventions/{id}/decision`                             | mock       |

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

- `apiFetch` lève `ApiError` (`status`, `code`, `retryAfterSeconds`, `fields`) pour toute réponse
  non 2xx. Chaque fonction mappe d'abord **ses** codes métier vers son `{ ok: false, error }`
  typé, puis délègue le reste à `toApiFailure` ; un code inconnu tombe sur un message générique,
  jamais sur le `message` brut de l'API.
- Codes de champ de validation (422) : `required`, `invalid`, `too_short`, `too_long`, `weak`,
  dans `ApiError.fields` (champ imbriqué : `address.city`). Exemple de table _code → message_ :
  `toSignupErrors` dans [actions/signup.ts](../src/app/actions/signup.ts).
- Statuts communs, traités une fois pour toutes dans [http.ts](../src/lib/api/http.ts) :

  | Statut | Traitement                                                                                        | Message (`apiFailureMessage`)      |
  | ------ | ------------------------------------------------------------------------------------------------- | ---------------------------------- |
  | 401    | Appel authentifié : `apiFetch` redirige vers `/connexion` (`redirectOnUnauthorized: false` sinon) | aucun                              |
  | 403    | `toApiFailure` → `forbidden`, journalisé (signe d'un bug : origine, CSRF ou rôle)                 | Action non autorisée               |
  | 413    | `toApiFailure` → `payload-too-large`                                                              | Fichier trop volumineux            |
  | 415    | `toApiFailure` → `unsupported-media-type`                                                         | Type de fichier refusé             |
  | 429    | `toApiFailure` → `too-many-attempts`, avec `retryAfterSeconds`                                    | Réessayer dans N minutes           |
  | autre  | `toApiFailure` → `unavailable`, journalisé                                                        | Service momentanément indisponible |

  Les résultats des fonctions incluent `ApiFailure` ([src/types/api.ts](../src/types/api.ts)) ;
  les actions affichent `apiFailureMessage(result)`
  ([form-state.ts](../src/lib/validation/form-state.ts)) pour ces cas. Attention : certains codes
  métier sont des 403 ou des 429 (`hero_request_not_eligible`, `account_locked`…), à traiter
  **avant** `toApiFailure`. Comme `redirect()` de Next lève une exception, un `catch` qui
  n'appelle pas `toApiFailure` doit appeler `unstable_rethrow(error)` en premier.

- Écritures authentifiées : `apiFetch` ajoute seul le `csrfToken` de `GET /auth/me` dans
  `X-CSRF-Token` dès qu'il reçoit un `sessionToken` sur une méthode autre que `GET`. La session
  est lue par `getApiSession`, mémorisée le temps d'un rendu : la page et ses écritures ne
  déclenchent qu'un seul `GET /auth/me`.

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

### Demandes de super-héros, comptes et types d'incident : choix du branchement

- **Demandes de super-héros** : une demande par compte, identifiée par `userId`. Le statut
  `pending | approved | rejected` correspond à `ModerationStatus` (`en-attente | valide | rejete`).
  Le contrat accepte un motif vide au refus ; **le frontend l'exige** (`validateRejectionReason`),
  car il est affiché au demandeur. Le citoyen dépose sa demande depuis « Mon compte » ; une fois
  acceptée, son rôle change aussitôt (le proxy l'envoie vers l'espace super-héros).
- **Comptes** : le contrat ne connaît pas « à contrôler / validé / rejeté », mais un état
  (`pending_activation | active | deactivated` → `AccountStatus`) et une vérification à part
  (`verifiedAt`, sans route pour l'annuler). Le tableau de bord ne compte donc plus les « comptes à
  contrôler » avec l'API (tuile masquée). Les **changements d'email** n'ont pas de route : la section
  est masquée avec l'API (`getEmailChangeRequests` renvoie `null`).
- **Types d'incident** : plus aucune liste en dur. Un incident embarque son type (`{ code, label }`,
  `typeCode` et `typeLabel` de l'API) ; les formulaires reçoivent la liste du référentiel et envoient
  son `id` (`typeId`). Le référentiel est **vide** dans une base neuve : tant qu'un modérateur n'a
  pas créé de type (`/admin/types-incident`), la déclaration d'incident est indisponible.

### Incidents, interventions et super-vilains : modèles différents

Le contrat existe désormais, mais les « Future implémentation » de `lib/api/` ont été écrites
avant lui et ne lui correspondent pas. Il faudra **réécrire les fonctions d'après `openapi.yaml`**,
pas seulement changer les URL.

| Sujet           | Frontend actuel (hypothèses)                                                             | Contrat                                                                                                                                          |
| --------------- | ---------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------ |
| Espaces         | Routes par rôle : `/hero/incidents…`, `/me/incidents`, `/admin/incidents…`               | Routes communes à tous les rôles (`/incidents`, `/interventions/{id}`) ; « mes incidents » = `GET /incidents?reporterId=…`                       |
| Intervention    | `POST` / `PUT /hero/incidents/{id}/intervention` : un héros se positionne sur l'incident | Ressource à part : `POST /incidents/{incidentId}/interventions`, puis `join`, invitations (`confirm` / `decline`), `outcome`, `outcome/validate` |
| Carte, priorité | `GET /incidents/map`, `GET /hero/incidents/map`, `PUT /hero/incidents/{id}/priority`     | Absents du contrat                                                                                                                               |
| Modération      | `PUT /admin/incidents/{id}/moderation`, `PATCH /admin/incidents/{id}`                    | Absente du contrat (seul l'arbitrage du résultat d'une intervention existe)                                                                      |
| Super-vilains   | Lecture seule : `GET /villains`, `/villains/{id}`, `/villains/{id}/incidents`            | `GET /villains` (recherche `q`), `POST /villains`, `PATCH /villains/{id}` ; ni détail ni incidents d'un vilain                                   |

À trancher avec le backend avant de brancher : ce qui manque au contrat (carte, priorité,
modération des incidents, fiche d'un vilain) est-il à ajouter, ou les écrans du frontend
doivent-ils changer ?

### Pagination

Les listes du contrat utilisent `limit` (1–100, défaut 20) et `offset`, et renvoient une page
(`HeroApplicantPage`, `UserSummaryPage`, `IncidentPage`…). Les fonctions `getXxx` du frontend
renvoient aujourd'hui des tableaux complets : en les branchant, leur passer un `PageRequest` et
leur faire renvoyer un `Page<T>` ([src/types/api.ts](../src/types/api.ts)), avec les outils de
[pagination.ts](../src/lib/api/pagination.ts) :

| Outil          | Rôle                                                                                          |
| -------------- | --------------------------------------------------------------------------------------------- |
| `pageQuery`    | Chaîne de requête : `limit`, `offset` ramenés dans les bornes de l'API, filtres non vides     |
| `fromApiPage`  | Page de l'API → `Page<T>`, avec la conversion de chaque élément                               |
| `toMockPage`   | Découpe une liste mockée comme l'API, pour garder la même signature avec et sans backend      |
| `pagePosition` | Page courante et nombre de pages, pour le composant de pagination des écrans d'administration |

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

Les formes de l'API sont générées depuis l'OpenAPI par
[`openapi-typescript`](https://openapi-ts.dev/) dans
[src/lib/api/schema.d.ts](../src/lib/api/schema.d.ts) (`npm run api:types`). Le fichier est
versionné : l'image Docker du frontend est construite sans le dossier `backend/`.

1. Une réponse brute de l'API se type avec `ApiSchema<"NomDuSchéma">` ([http.ts](../src/lib/api/http.ts)),
   **jamais** avec une interface écrite à la main : le compilateur vérifie alors chaque
   conversion.
2. `src/types/` reste la forme « frontend » (rôles, statuts en français). Les fonctions de
   `lib/api/` branchées convertissent la forme de l'API vers ces types ; **les composants ne
   manipulent jamais la forme brute**.
3. Après toute modification de `backend/openapi/openapi.yaml` : `npm run api:types`, puis
   corriger ce que `tsc` signale. La CI du frontend (lancée aussi quand le contrat change)
   échoue si `schema.d.ts` n'est pas à jour (`npm run api:types:check`).
4. Une route ajoutée au backend sans contrat est refusée par ses propres tests : le frontend
   n'a donc pas à anticiper de route absente de `openapi.yaml`.

## Checklist de branchement d'une fonction

Modèle à suivre : [src/lib/api/auth.ts](../src/lib/api/auth.ts) (`login`, `getSessionUser`,
`logout`).

1. La route existe-t-elle dans `openapi.yaml` ? Sinon, la faire ajouter côté backend d'abord.
2. Garder la signature ; brancher sur `isApiEnabled` : `apiFetch(…)` si l'API est configurée,
   le mock sinon (l'application doit toujours démarrer sans backend).
3. Passer le jeton dans `sessionToken` (le jeton CSRF est ajouté seul aux écritures).
4. Typer la réponse brute avec `ApiSchema<"…">` et la convertir (rôles, statuts, dates) vers les
   types de `src/types/`. Liste : `PageRequest` en entrée, `Page<T>` en sortie (`pageQuery`,
   `fromApiPage`, `toMockPage` pour le mock).
5. Mapper les codes métier (`ApiError.code`) vers le résultat `{ ok: false, error }`, puis
   `return toApiFailure(error, "…")` ; ajouter `ApiFailure` au type du résultat et afficher
   `apiFailureMessage` dans l'action.
6. Mettre à jour le tableau de la section 1. Quand le domaine n'utilise plus `userFromToken`,
   vérifier si `asMockDataUser` sert encore.
